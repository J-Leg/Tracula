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

	ROUTINELIMIT = 50 // Max number of go-routines running concurrently
)

// Execute : Core execution for daily updates
// Update all apps
func Execute(cfg *Config) {
	cfg.Trace.Debug.Printf("initiate daily execution.")

	appList, err := cfg.GetAppList()
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
			go processApp(cfg, appList[j], workChannel)
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
	appList, err := cfg.GetAppList()
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
			go processAppMonthly(cfg, appList[j], &currentDateTime, workChannel)
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

func finalise(t int, numSuccess, numError *int, ch chan<- bool, bar *pb.ProgressBar, cfg *Config) {
	close(ch)

	if cfg.LocalEnabled {
		bar.Finish()
	}

	var jobType string
	if t == DAILY {
		jobType = "daily"
	} else if t == MONTHLY {
		jobType = "monthly"
	} else if t == RECOVERY {
		jobType = "recovery"
	} else {
		cfg.Trace.Error.Printf("Invalid job type %d", t)
	}

	cfg.Trace.Info.Printf("%s execution REPORT:\n    success: %d\n    errors: %d", jobType, *numSuccess, *numError)
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

	var numExceptions = len(*appsToUpdate)
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
			go processApp(cfg, (*appsToUpdate)[j], workChannel)
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

func processAppMonthly(
	cfg *Config,
	app AppShadow,
	currentDateTime *time.Time,
	ch chan<- bool) {
	cfg.Trace.Debug.Printf("monthly process on app: %s - ID: %+v.", app.Ref.Name, app.Ref.ID)

	var err error
	defer workDone(ch, &err)

	appBom, err := cfg.GetApp(app.Ref.ID)
	if err != nil {
		cfg.Trace.Error.Printf("Error retrieving daily metric: %s", err)
		return
	}

	newDailyMetricList, newPeak, newAverage := monthlySanitise(appBom, currentDateTime)
	appBom.DailyMetrics = *newDailyMetricList
	cfg.Trace.Debug.Printf("Computed monthly average %d, for app %s", newAverage, app.Ref.Name)

	sortDates(&appBom.Metrics)
	previousMonthMetrics := appBom.Metrics[len(appBom.Metrics)-1]

	cfg.Trace.Info.Printf("Construct month element: month - %s, year - %d", currentDateTime.Month().String(), currentDateTime.Year())

	newMonth := constructNewMonthMetric(&previousMonthMetrics, newPeak, newAverage, currentDateTime)
	appBom.Metrics = append(appBom.Metrics, *newMonth)

	err = cfg.UpdateApp(appBom)
	if err != nil {
		cfg.Trace.Error.Printf("Error updating app %s. %s", app.Ref.Name, err)
		return
	}

	cfg.Trace.Debug.Printf("monthly process success for app: %s - ID: %+v.", app.Ref.Name, app.Ref.ID)
	return
}

func processApp(cfg *Config, app AppShadow, ch chan<- bool) {
	cfg.Trace.Debug.Printf("daily process on app: %s - id: %+v.", app.Ref.Name, app.Ref.ID)

	var err error
	defer workDone(ch, &err)

	population, err := stats.Fetch(app.Ref.Domain, app.Ref.DomainID)
	if err != nil {
		cfg.Trace.Error.Printf("Error fetch population for app %s! %s", app.Ref.Name, err)
		err = cfg.PushException(&app)
		if err != nil {
			cfg.Trace.Error.Printf("error inserting app %d to exception queue! %s", app.Ref.DomainID, err)
		}
		return
	}

	err = cfg.PushDaily(app.Ref.ID, &DailyMetric{Date: app.Date, PlayerCount: population})
	if err != nil {
		err = cfg.PushException(&app)
		if err != nil {
			cfg.Trace.Error.Printf("error inserting app %d to exception queue! %s", app.Ref.DomainID, err)
		}
		return
	}
	cfg.Trace.Debug.Printf("daily process success on app: %s - id: %+v.", app.Ref.Name, app.Ref.ID)
	return
}

func workDone(ch chan<- bool, err *error) {
	if *err == nil {
		ch <- true
	} else {
		ch <- false
	}
}
