package tracula

import (
	"github.com/J-Leg/tracula/internal/stats"
	"github.com/cheggaaa/pb/v3"
	"math"
	"os"
	"time"
)

// Constants
const (
	MONTHS           = 12
	FUNCTIONDURATION = 8

	DAILY    = 0
	MONTHLY  = 1
	RECOVERY = 2
	REFRESH  = 3
	TRACKER  = 4

	ROUTINELIMIT        = 50 // Max number of go-routines running concurrently
	REFRESHROUTINELIMIT = 50

	NOACTIVITYLIMIT = 3
)

// Execute : Core execution for daily updates
// Update all apps
func Execute(cfg *Config) {
	cfg.Trace.Debug.Printf("initiate daily execution.")

	appList, err := cfg.getAppListTracked()
	if err != nil {
		cfg.Trace.Error.Printf("failed to retrieve app list. %s", err)
		return
	}

	workChannel := make(chan bool)
	timeout := time.After(FUNCTIONDURATION * time.Minute)

	var numSuccess, numErrors int = 0, 0
	var numApps int = len(appList)
	var numBatches int = int(math.Ceil(float64(numApps / ROUTINELIMIT)))

	var bar *pb.ProgressBar
	if cfg.LocalEnabled {
		bar = pb.StartNew(numApps)
		bar.SetRefreshRate(time.Second)
		bar.SetWriter(os.Stdout)
		bar.Start()
	}

	defer finalise(DAILY, &numSuccess, &numErrors, workChannel, bar, cfg)

	for i := 0; i <= numBatches; i++ {
		startIdx := i * ROUTINELIMIT
		endIdx := min(startIdx+ROUTINELIMIT, numApps)

		for j := startIdx; j < endIdx; j++ {
			go processApp(cfg, &appList[j], workChannel)
		}

		// Wait on communication received from each go routine
		// For each item in the batch
		for j := startIdx; j < endIdx; j++ {
			select {
			case msg := <-workChannel:
				if msg {
					numSuccess++
				} else {
					numErrors++
				}
			case <-timeout:
				cfg.Trace.Info.Printf("Daily process runtime exceeding maximum duration.")
				return
			}
			if cfg.LocalEnabled {
				bar.Increment()
			}
		}
	}
	return
}

// ExecuteMonthly : Monthly process
func ExecuteMonthly(cfg *Config) {
	cfg.Trace.Debug.Printf("initiate monthly execution.")
	appList, err := cfg.getAppListFull()
	if err != nil {
		cfg.Trace.Error.Printf("failed to retrieve app list. %s", err)
		return
	}

	var currentDateTime time.Time = time.Now()

	workChannel := make(chan bool)
	timeout := time.After(FUNCTIONDURATION * time.Minute)

	var numApps int = len(appList)
	var numBatches int = int(math.Ceil(float64(numApps / ROUTINELIMIT)))
	var numSuccess, numErrors int = 0, 0

	var bar *pb.ProgressBar
	if cfg.LocalEnabled {
		bar = pb.StartNew(numApps)
		bar.SetRefreshRate(time.Second)
		bar.SetWriter(os.Stdout)
		bar.Start()
	}

	defer finalise(MONTHLY, &numSuccess, &numErrors, workChannel, bar, cfg)

	for i := 0; i <= numBatches; i++ {
		startIdx := i * ROUTINELIMIT
		endIdx := min(startIdx+ROUTINELIMIT, numApps)

		for j := startIdx; j < endIdx; j++ {
			go processAppMonthly(cfg, &appList[j], &currentDateTime, workChannel)
		}

		// Wait on communication received from each go routine
		// For each item in the batch
		for j := startIdx; j < endIdx; j++ {
			select {
			case msg := <-workChannel:
				if msg {
					numSuccess++
				} else {
					numErrors++
				}
			case <-timeout:
				cfg.Trace.Info.Printf("Monthly process runtime exceeding maximum duration.")
				return
			}
			if cfg.LocalEnabled {
				bar.Increment()
			}
		}
	}
	return
}

// ExecuteRefresh updates the app library
func ExecuteRefresh(cfg *Config) {
	appList, err := cfg.getAppListMonthData()
	if err != nil {
		cfg.Trace.Error.Printf("error retrieving app list %s", err)
		return
	}
	// Convert list to map
	var currentAppMap map[int]bool = make(map[int]bool)
	for _, appElement := range appList {
		currentAppMap[appElement.StaticData.AppID] = true
	}

	newDomainAppMap, err := stats.FetchApps()
	cfg.Trace.Info.Printf("lol %+v", newDomainAppMap)
	if err != nil {
		cfg.Trace.Error.Printf("error fetching latest apps %s", err)
		return
	}
	// Identify and construct new apps
	var newApps []App
	for domain, appMap := range newDomainAppMap {
		for appId, appName := range appMap {

			// Check if exists already in library
			_, ok := currentAppMap[appId]
			if ok {
				continue
			}

			cfg.Trace.Info.Printf("New app: %s - id: %d", appName, appId)

			newStaticData := StaticAppData{Name: appName, AppID: appId, Domain: domain}
			newApp := App{
				Metrics:      make([]Metric, 0), // Initialise 0 len slice instead of nil slice
				DailyMetrics: make([]DailyMetric, 0),
				StaticData:   newStaticData,
			}
			newApps = append(newApps, newApp)
		}
	}

	workChannel := make(chan bool)
	timeout := time.After(FUNCTIONDURATION * time.Minute)

	var numUpdates int = len(newApps)
	var numBatches int = int(math.Ceil(float64(numUpdates / ROUTINELIMIT)))
	var numSuccess, numErrors int = 0, 0

	var bar *pb.ProgressBar
	if cfg.LocalEnabled {
		bar = pb.StartNew(numUpdates)
		bar.SetRefreshRate(time.Second)
		bar.SetWriter(os.Stdout)
		bar.Start()
	}
	defer finalise(REFRESH, &numSuccess, &numErrors, workChannel, bar, cfg)

	for i := 0; i <= numBatches; i++ {
		startIdx := i * ROUTINELIMIT
		endIdx := min(startIdx+ROUTINELIMIT, numUpdates)

		for j := startIdx; j < endIdx; j++ {
			go processRefresh(cfg, &(newApps[j]), workChannel)
		}

		// Wait on communication received from each go routine (in the batch)
		for j := startIdx; j < endIdx; j++ {
			select {
			case msg := <-workChannel:
				if msg {
					numSuccess++
				} else {
					numErrors++
				}

			case <-timeout:
				cfg.Trace.Info.Printf("Refresh process runtime exceeding maximum duration.")
				return
			}
			if cfg.LocalEnabled {
				bar.Increment()
			}
		}
	}
	return
}

// ExecuteRecovery : Best effort to retry all exception instances
func ExecuteRecovery(cfg *Config) {
	var appsToUpdate, err = cfg.GetExceptions()
	if err != nil {
		cfg.Trace.Error.Printf("Error retrieving exceptions. %s", err)
		return
	}

	cfg.FlushExceptions()

	workChannel := make(chan bool)
	timeout := time.After(FUNCTIONDURATION * time.Minute)

	var numExceptions = len(appsToUpdate)
	var numBatches int = int(math.Ceil(float64(numExceptions / ROUTINELIMIT)))
	var numSuccess, numErrors int = 0, 0

	var bar *pb.ProgressBar
	if cfg.LocalEnabled {
		bar = pb.StartNew(numExceptions)
		bar.SetRefreshRate(time.Second)
		bar.SetWriter(os.Stdout)
		bar.Start()
	}

	defer finalise(RECOVERY, &numSuccess, &numErrors, workChannel, bar, cfg)

	for i := 0; i <= numBatches; i++ {
		startIdx := i * ROUTINELIMIT
		endIdx := min(startIdx+ROUTINELIMIT, numExceptions)

		for j := startIdx; j <= endIdx; j++ {
			go processApp(cfg, &appsToUpdate[j], workChannel)
		}

		for j := startIdx; j < endIdx; j++ {
			select {
			case msg := <-workChannel:
				if msg {
					numSuccess++
				} else {
					numErrors++
				}
			case <-timeout:
				cfg.Trace.Info.Printf("Daily process runtime exceeding maximum duration.")
				return
			}

			if cfg.LocalEnabled {
				bar.Increment()
			}
		}
	}
	close(workChannel)
	cfg.Trace.Info.Printf("[Exceptions] recovery process complete.")
	return
}

// ExecuteTracker runs a job to aggregate any apps that are worth tracking
func ExecuteTracker(cfg *Config) {
	err := cfg.flushTrackPool()
	if err != nil {
		cfg.Trace.Error.Printf("[Tracker] error flushing pool. %s", err)
		return
	}

	appList, err := cfg.getAppListMonthData()
	if err != nil {
		cfg.Trace.Error.Printf("error retrieving app list: %s", err)
		return
	}

	workChannel := make(chan bool)
	timeout := time.After(FUNCTIONDURATION * time.Minute)

	var numApps int = len(appList)
	var numBatches int = int(math.Ceil(float64(numApps / REFRESHROUTINELIMIT)))
	var numSuccess, numErrors int = 0, 0

	var bar *pb.ProgressBar
	if cfg.LocalEnabled {
		bar = pb.StartNew(numApps)
		bar.SetRefreshRate(time.Second)
		bar.SetWriter(os.Stdout)
		bar.Start()
	}

	defer finalise(TRACKER, &numSuccess, &numErrors, workChannel, bar, cfg)

	for i := 0; i <= numBatches; i++ {
		startIdx := i * ROUTINELIMIT
		endIdx := min(startIdx+ROUTINELIMIT, numApps)

		for j := startIdx; j < endIdx; j++ {
			go processAppTrack(cfg, &appList[j], workChannel)
		}

		// Wait on communication received from each go routine
		// For each item in the batch
		for j := startIdx; j < endIdx; j++ {
			select {
			case msg := <-workChannel:
				if msg {
					numSuccess++
				} else {
					numErrors++
				}
			case <-timeout:
				cfg.Trace.Info.Printf("Tracker process runtime exceeding maximum duration.")
				return
			}
			if cfg.LocalEnabled {
				bar.Increment()
			}
		}
	}
}

func processAppTrack(cfg *Config, appBom *App, ch chan<- bool) {
	var err error
	defer workDone(ch, &err)

	// Add to the tracker pool if satisfies the below condition
	// A non-zero player count over the last 3 months (or up to 3 months)
	var monthMetricList []Metric = appBom.Metrics
	var isWorthTracking bool = false
	for i := len(monthMetricList) - 1; i >= max(0, len(monthMetricList)-1-NOACTIVITYLIMIT); i-- {
		if monthMetricList[i].AvgPlayers > 0 {
			isWorthTracking = true
			break
		}
	}

	if !isWorthTracking {
		val, err := stats.Fetch(appBom.StaticData.Domain, appBom.StaticData.AppID)
		if err != nil {
			cfg.Trace.Error.Printf("[TRACKER] Error fetching: %s", err)
		}

		if val == 0 {
			return
		}
	}
	err = cfg.pushTrackedApp(appBom)
	if err != nil {
		cfg.Trace.Error.Printf("[TRACKER] error pushing new tracked app: %s", err)
		return
	}
	cfg.Trace.Debug.Printf("[TRACKER] Added app: %s to track pool.", appBom.StaticData.Name)
}

func processAppMonthly(
	cfg *Config,
	app *App,
	currentDateTime *time.Time,
	ch chan<- bool) {
	cfg.Trace.Debug.Printf("monthly process on app: %s - ID: %+v.", app.StaticData.Name, app.ID)

	var err error
	defer workDone(ch, &err)

	newPeak, newAverage := monthlySanitise(app, currentDateTime)
	cfg.Trace.Debug.Printf("Computed monthly average %d, for app %s", newAverage, app.StaticData.Name)

	sortDates(app.Metrics)
	var previousMonthMetricsPtr *Metric = nil
	if len(app.Metrics) > 0 {
		previousMonthMetricsPtr = &app.Metrics[len(app.Metrics)-1]
	}
	cfg.Trace.Info.Printf("Construct month element: month - %s, year - %d", currentDateTime.Month().String(), currentDateTime.Year())

	newMonth := constructNewMonthMetric(previousMonthMetricsPtr, newPeak, newAverage, currentDateTime)
	app.Metrics = append(app.Metrics, *newMonth)

	err = cfg.UpdateApp(app)
	if err != nil {
		cfg.Trace.Error.Printf("Error updating app %s. %s", app.StaticData.Name, err)
		return
	}

	cfg.Trace.Debug.Printf("monthly process success for app: %s - ID: %+v.", app.StaticData.Name, app.ID)
	return
}

func processApp(cfg *Config, app *AppRef, ch chan<- bool) {
	appData := app.StaticData
	cfg.Trace.Debug.Printf("daily process on app: %s - id: %+v.", appData.Name, app.RefID)

	var err error
	defer workDone(ch, &err)

	currentDateTime, err := time.Parse(DATEPATTERN, time.Now().UTC().String()[:19])
	if err != nil {
		cfg.Trace.Error.Printf("unable to construct datetime: %s", err)
		return
	}

	population, err := stats.Fetch(appData.Domain, appData.AppID)
	if err != nil {
		cfg.Trace.Error.Printf("Error fetch population for app %s! %s", appData.Name, err)
		err = cfg.PushException(app, &currentDateTime)
		if err != nil {
			cfg.Trace.Error.Printf("error inserting app %d to exception queue! %s", app.RefID, err)
		}
		return
	}

	err = cfg.PushDaily(app.RefID, &DailyMetric{Date: currentDateTime, PlayerCount: population})
	if err != nil {
		err = cfg.PushException(app, &currentDateTime)
		if err != nil {
			cfg.Trace.Error.Printf("error inserting app %d to exception queue! %s", app.RefID, err)
		}
		return
	}
	cfg.Trace.Debug.Printf("daily process success on app: %s - id: %+v.", app.StaticData.Name, app.RefID)
	return
}

func processRefresh(cfg *Config, app *App, ch chan<- bool) {
	var err error
	defer workDone(ch, &err)

	err = cfg.PushApp(app)
	if err != nil {
		cfg.Trace.Error.Printf("Error inserting new app: %s", app.StaticData.Name)
		return
	}

	cfg.Trace.Debug.Printf("refresh process success for app: %s - ID: %+v", app.StaticData.Name, app.StaticData.AppID)
	return
}

func workDone(ch chan<- bool, err *error) {
	if *err == nil {
		ch <- true
	} else {
		ch <- false
	}
}
