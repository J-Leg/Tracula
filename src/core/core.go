package core

import (
	"context"
	"github.com/cheggaaa/pb/v3"
	"math"
	"os"
	"playercount/src/db"
	"playercount/src/env"
	"playercount/src/stats"
	"time"
)

// Constants
const (
	MONTHS = 12
)

// Execute : Core execution for daily updates
// Update all apps
func Execute(ctx context.Context) {
	var cfg *env.Config = env.InitConfig(ctx)
	var dbcfg db.Dbcfg = db.Dbcfg(*cfg)
	cfg.Trace.Debug.Printf("initiate daily execution.\n")

	appList, err := dbcfg.GetAppList()
	if err != nil {
		return
	}

	bar := pb.StartNew(len(appList))
	bar.SetRefreshRate(time.Second)
	bar.SetWriter(os.Stdout)
	bar.Start()

	for _, app := range appList {

		// Update progress
		bar.Increment()
		time.Sleep(time.Millisecond)
		err := processApp(&dbcfg, &app)
		if err != nil {
			continue
		}
	}
	bar.Finish()
	cfg.Trace.Debug.Printf("conclude daily execution.\n")

	// Close
	cfg.LoggerClient.Close()
	return
}

// ExecuteMonthly : Monthly process
func ExecuteMonthly(ctx context.Context) {
	var cfg *env.Config = env.InitConfig(ctx)
	var dbcfg db.Dbcfg = db.Dbcfg(*cfg)
	cfg.Trace.Debug.Printf("initiate monthly execution.\n")

	appList, err := dbcfg.GetAppList()
	if err != nil {
		cfg.Trace.Error.Printf("failed to retrieve app list. %s", err)
		return
	}

	bar := pb.StartNew(len(appList))
	bar.SetRefreshRate(time.Second)
	bar.SetWriter(os.Stdout)
	bar.Start()

	currentDateTime := time.Now()
	var monthToAvg time.Month = currentDateTime.Month() - 1
	if currentDateTime.Month() == 1 {
		monthToAvg = 12
	}

	var yearToAvg int = currentDateTime.Year()
	if currentDateTime.Month() == 1 {
		yearToAvg = currentDateTime.Year() - 1
	}

	for _, app := range appList {
		// Update progress
		bar.Increment()
		time.Sleep(time.Millisecond)

		processAppMonthly(&dbcfg, &app, monthToAvg, yearToAvg)
	}
	bar.Finish()
	cfg.Trace.Debug.Printf("conclude monthly execution.\n")

	// Close
	cfg.LoggerClient.Close()
	return
}

// ExecuteRecovery : Best effort to retry all exception instances
func ExecuteRecovery(ctx context.Context) {
	var cfg *env.Config = env.InitConfig(ctx)
	var dbcfg db.Dbcfg = db.Dbcfg(*cfg)
	var appsToUpdate, err = dbcfg.GetExceptions()
	if err != nil {
		cfg.Trace.Error.Printf("Error retrieving exceptions. %s", err)
		return
	}

	dbcfg.FlushExceptions()

	cfg.Trace.Info.Printf("[Exceptions] re-do daily process.\n")
	for _, app := range *appsToUpdate {
		err = processApp(&dbcfg, &app)
		if err != nil {
			cfg.Trace.Error.Printf("Daily retry (%s) failed for app: %+v - %s", app.Date, app.Ref.ID, err)
			continue
		}
	}

	cfg.Trace.Info.Printf("[Exceptions] recovery process complete.\n")
	cfg.LoggerClient.Close()
	return
}

func processAppMonthly(cfg *db.Dbcfg, app *db.AppShadow, monthToAvg time.Month, yearToAvg int) {
	cfg.Trace.Debug.Printf("monthly process on app: %s - id: %+v.\n", app.Ref.Name, app.Ref.ID)
	dailyMetricList, err := cfg.GetDailyList(app.Ref.ID)
	if err != nil {
		cfg.Trace.Error.Printf("Error retrieving daily metric: %s", err)
		return
	}
	// Initialise a new list
	var newDailyMetricList []stats.DailyMetric

	var total int = 0
	var numCounted int = 0
	var newPeak float64 = 0

	for _, dailyMetric := range *dailyMetricList {
		var elementMonth = dailyMetric.Date.Month()
		var elementYear = dailyMetric.Date.Year()
		var monthDiff = monthToAvg - elementMonth

		// Only keep daily metrics up to the last 3 months
		// This condition "should" be enough if older months were correctly purged
		if (monthDiff+MONTHS)%MONTHS < 3 {

			// Secondary conidition is just for assurance
			if monthDiff < 0 && (elementYear != yearToAvg-1) {
				continue
			}

			newDailyMetricList = append(newDailyMetricList, dailyMetric)
		}

		if elementMonth != monthToAvg {
			continue
		}

		newPeak = math.Max(newPeak, float64(dailyMetric.PlayerCount))
		total += dailyMetric.PlayerCount
		numCounted++
	}

	var newAverage int = 0
	if numCounted > 0 {
		newAverage = total / numCounted
	}

	cfg.Trace.Debug.Printf("Computed average player count of: %d on month: %d using %d dates.\n",
		newAverage, monthToAvg, numCounted)

	err = cfg.UpdateDailyList(app.Ref.ID, &newDailyMetricList)
	if err != nil {
		cfg.Trace.Error.Printf("Error updating daily metric list: %s.\n", err)
		return
	}

	var monthMetricListPtr *[]db.Metric
	monthMetricListPtr, err = cfg.GetMonthlyList(app.Ref.ID)
	if err != nil {
		cfg.Trace.Error.Printf("Error retrieving month metrics: %s.\n", err)
		return
	}

	monthSort(monthMetricListPtr)

	monthMetricList := *monthMetricListPtr
	previousMonthMetrics := &monthMetricList[len(monthMetricList)-1]
	newMonth := constructNewMonthMetric(previousMonthMetrics, newPeak, float64(newAverage), monthToAvg, yearToAvg)
	monthMetricList = append(monthMetricList, *newMonth)

	cfg.UpdateMonthlyList(app.Ref.ID, monthMetricListPtr)
	if err != nil {
		cfg.Trace.Error.Printf("error updating month metric list %s.\n", err)
		return
	}

	cfg.Trace.Debug.Printf("monthly process success.\n")
	return
}

func processApp(cfg *db.Dbcfg, app *db.AppShadow) error {
	cfg.Trace.Debug.Printf("daily process on app: %s - id: %+v.\n", app.Ref.Name, app.Ref.ID)
	dm, err := stats.Fetch(app.Date, app.Ref.Domain, app.Ref.DomainID)
	if err != nil {
		err = cfg.PushException(app)
		if err != nil {
			cfg.Trace.Error.Printf("error inserting app %d to exception queue! %s\n", app.Ref.DomainID, err)
			// What do?
		}
		return err
	}

	err = cfg.PushDaily(app.Ref.ID, dm)
	if err != nil {
		err = cfg.PushException(app)
		if err != nil {
			cfg.Trace.Error.Printf("error inserting app %d to exception queue! %s\n", app.Ref.DomainID, err)
			// What do?
		}
		return err
	}
	cfg.Trace.Debug.Printf("daily process success on app: %s - id: %+v.\n", app.Ref.Name, app.Ref.ID)
	return nil
}
