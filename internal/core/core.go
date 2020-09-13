package core 

import (
  "github.com/j-leg/tracula/config"
  "github.com/j-leg/tracula/internal/db"
  "github.com/j-leg/tracula/internal/stats"
  "context"
  "time"
  "github.com/cheggaaa/pb/v3"
  "math"
  "os"
)

// Constants
const (
  MONTHS           = 12
  FUNCTIONDURATION = 8

  ROUTINELIMIT        = 50 // Max number of go-routines running concurrently
  REFRESHROUTINELIMIT = 50
  DATEPATTERN         = "2006-01-02 15:04:05"

  NOACTIVITYLIMIT   = 3
  LIMIT             = 50 
)

// Exported entry points
// Daily
func Daily(cfg *config.Config) {
  execute(cfg, db.DAILY, dailyAtomic)
}

// Monthly
func Monthly(cfg *config.Config) {
  execute(cfg, db.MONTHLY, monthlyAtomic)
}

// Track
func Track(cfg *config.Config) {
  execute(cfg, db.TRACK, trackAtomic)
}

// Recover
func Recover(cfg *config.Config) {
  // TODO
  db.Flush(cfg.Ctx, cfg.Col.Exceptions)
  execute(cfg, db.RECOVERY, dailyAtomic)
}

// Refresh - TODO
func Refresh(cfg *config.Config) {
  appList, err := db.GetFullStaticData(cfg.Ctx, cfg.Col.Stats)
  if err != nil {
    cfg.Trace.Error.Printf("error retrieving app list %s", err)
    return
  }
  // Convert list to map
  var currentAppMap map[int]bool = make(map[int]bool)
  for _, appElement := range appList {
    currentAppMap[appElement.AppID] = true
  }

  newDomainAppMap, err := stats.FetchApps()
  if err != nil {
    cfg.Trace.Error.Printf("error fetching latest apps %s", err)
    return
  }
  // Identify and construct new apps
  var newApps []db.App
  for domain, appMap := range newDomainAppMap {
    for appId, appName := range appMap {

      // Check if exists already in library
      _, ok := currentAppMap[appId]
      if ok { continue }

      cfg.Trace.Info.Printf("New app: %s - id: %d", appName, appId)

      newStaticData := db.StaticAppData{Name: appName, AppID: appId, Domain: domain}
      newApp := db.App{
        Metrics:      make([]db.Metric, 0), // Initialise 0 len slice instead of nil slice
        DailyMetrics: make([]db.DailyMetric, 0),
        StaticData:   newStaticData,
      }
      newApps = append(newApps, newApp)
    }
  }
  // TODO
  execute(cfg, db.REFRESH, refreshAtomic)
}

type executeAtomic func(ctx context.Context, app *db.App, cols *config.Collections, ch chan<-msgAtomic)  

func execute(cfg *config.Config, jobType int, atomic executeAtomic) {
  numDocuments, cursor, err := db.GetJobParams(cfg, jobType)
  if err != nil {
    cfg.Trace.Error.Printf("Error initialising job params: %s", err)
    return
  }

  numBatches := int(math.Ceil(float64(numDocuments/LIMIT)))
  numSuccess, numErrors := 0, 0

  // Local - only
  var bar *pb.ProgressBar
  if cfg.LocalEnabled {
    bar = pb.StartNew(numDocuments)
    bar.SetRefreshRate(time.Second)
    bar.SetWriter(os.Stdout)
    bar.Start()
  }

  workChannel := make(chan msgAtomic)
  timeout := time.After(FUNCTIONDURATION * time.Minute)

  for i := 0; i <= numBatches; i++ {
    curr := 0
    numRoutines := 0
    for curr < LIMIT && cursor.Next(cfg.Ctx) {
      var app db.App
      curr++
      if err := cursor.Decode(&app); err != nil {
        cfg.Trace.Error.Printf("Error decoding. %s", err)
        continue
      }
      
      // Deferred cancel handled in atomic go-routine
      childCtx, _ := context.WithCancel(cfg.Ctx)
      go atomic(childCtx, &app, cfg.Col, workChannel)
      numRoutines++
    }

    for completed := 0; completed < numRoutines; completed++ {
      select {
      case msg := <- workChannel:
        if msg.err == nil {
          cfg.Trace.Debug.Printf("Successful process [%d] for app %s.", jobType, msg.ID)
          numSuccess++
        } else {
          cfg.Trace.Error.Printf("Error process [%d] app %s - %s", jobType, msg.ID, msg.err.Error()) 
          numErrors++
        }
      case <- timeout:
        cfg.Trace.Info.Println("Process timeout signal received. Terminate.")
        close(workChannel)
        cursor.Close(cfg.Ctx)
        return
      }
      if cfg.LocalEnabled { bar.Increment() }
    }
  }

  close(workChannel)
  cursor.Close(cfg.Ctx)
  if bar != nil { bar.Finish() }

  var job string
  switch jobType {
    case db.DAILY:
      job = "daily"
    case db.MONTHLY:
      job = "monthly"
    case db.RECOVERY:
      job = "recovery"
    case db.REFRESH:
      job = "refresh"
    case db.TRACK:
      job = "track"
    default:
      cfg.Trace.Error.Printf("Invalid job type %d", jobType)
  }
  cfg.Trace.Info.Printf("%s execution REPORT:\n    success: %d\n    errors: %d", job, numSuccess, numErrors)
}

func dailyAtomic(ctx context.Context, app *db.App, cols *config.Collections, ch chan<-msgAtomic) {
  var err error
  defer finaliseAtomic(ctx, ch, app.ID.String(), &err)

  var currDateTime time.Time
  currDateTime, err = time.Parse(DATEPATTERN, time.Now().UTC().String()[:19])
  if err != nil { return }

  var quantity int
  quantity, err = stats.Fetch(app.StaticData.Domain, app.StaticData.AppID)
  if err != nil { return }

  newDailyElement := db.DailyMetric{Date: currDateTime, PlayerCount: quantity}
  app.DailyMetrics = append(app.DailyMetrics, newDailyElement)
  app.LastMetric = newDailyElement
  
  err = db.UpdateApp(ctx, app, cols.Stats)
}

func monthlyAtomic(ctx context.Context, app *db.App, cols *config.Collections, ch chan<-msgAtomic) {
  var err error
  defer finaliseAtomic(ctx, ch, app.ID.String(), &err)

  var currDateTime time.Time
  currDateTime, err = time.Parse(DATEPATTERN, time.Now().UTC().String()[:19])
  if err != nil { return }
  
  newPeak, newAverage := analyseMonthData(app, &currDateTime)
  
  var prevMonthMetricPtr *db.Metric = nil
  if len(app.Metrics) > 0 {
    prevMonthMetricPtr = &(app.Metrics[len(app.Metrics)-1])
  }

  newMonthMetricPtr := constructNewMonthMetric(prevMonthMetricPtr, newPeak, newAverage, &currDateTime)
  app.Metrics = append(app.Metrics, *newMonthMetricPtr)

  err = db.UpdateApp(ctx, app, cols.Stats)
}

func refreshAtomic(ctx context.Context, app *db.App, cols *config.Collections, ch chan<-msgAtomic) {
  var err error
  defer finaliseAtomic(ctx, ch, app.ID.String(), &err)

  err = db.AddNewApp(ctx, app, cols.Stats)
}

func trackAtomic(ctx context.Context, app *db.App, cols *config.Collections, ch chan<-msgAtomic) {
  var err error
  defer finaliseAtomic(ctx, ch, app.ID.String(), &err)

  // Set track flag
  // A non-zero playercount over the last 3 months (or up to 3 months)
  var monthMetricList []db.Metric = app.Metrics
  var isWorthTracking bool = false
  for i := len(monthMetricList) - 1; i >= max(0, len(monthMetricList)-1-NOACTIVITYLIMIT); i-- {
    if monthMetricList[i].AvgPlayers > 0 {
      isWorthTracking = true
      break
    }
  }

  if !isWorthTracking {
    var val int
    val, err = stats.Fetch(app.StaticData.Domain, app.StaticData.AppID)
    if err != nil { return }
    if val == 0 {
      if app.Tracked { db.SetTrackFlag(ctx, app.ID, false, cols.Stats) }
      return
    }
  }
  if !app.Tracked { db.SetTrackFlag(ctx, app.ID, true, cols.Stats) }
}
