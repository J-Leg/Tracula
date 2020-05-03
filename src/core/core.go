package core

import (
	"cloud.google.com/go/logging"
	"context"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"math"
	"os"
	"playercount/src/db"
	"playercount/src/env"
	"playercount/src/stats"
	"time"
)

const monthsInYear = 12

// Execute : Core execution for daily updates
// Update all apps
func Execute() {
	var appList = db.GetAppList()

	bar := pb.StartNew(len(appList))
	bar.SetRefreshRate(time.Second)
	bar.SetWriter(os.Stdout)
	bar.Start()

	for _, app := range appList {

		// Update progress
		bar.Increment()
		time.Sleep(time.Millisecond)
		err := processApp(&app)
		if err != nil {
			continue
		}
	}
	bar.Finish()
}

// ExecuteMonthly : Monthly process
func ExecuteMonthly(ctx context.Context) {
	var cfg *env.Config = env.InitConfig(ctx)

	var appList = db.GetAppList()
	bar := pb.StartNew(len(appList))
	bar.SetRefreshRate(time.Second)
	bar.SetWriter(os.Stdout)
	bar.Start()

	for _, app := range appList {

		// Update progress
		bar.Increment()
		time.Sleep(time.Millisecond)
		err := processAppMonthly(&app)
		if err != nil {
			continue
		}
	}
	bar.Finish()
}

func processAppMonthly(app *db.AppRef) error {
	dailyMetricList, err := db.GetDailyMetricList(app.Ref.ID)
	if err != nil {
		return err
	}

	currentDateTime := time.Now()
	var monthToAvg time.Month = currentDateTime.Month() - 1
	if currentDateTime.Month() == 1 {
		monthToAvg = 12
	}

	var yearToAvg int = currentDateTime.Year()
	if currentDateTime.Month() == 1 {
		yearToAvg = currentDateTime.Year() - 1
	}

	// Initialise a new list
	var newDailyMetricList []stats.DailyMetric

	var total int = 0
	var numCounted int = 0
	var newPeak float64 = 0

	for _, dailyMetric := range *dailyMetricList {
		var elementMonth = dailyMetric.Date.Month()

		// Only keep daily metrics for the last 3 months
		if (monthToAvg-elementMonth)%monthsInYear < 3 {
			newDailyMetricList = append(newDailyMetricList, dailyMetric)
		}

		if elementMonth != monthToAvg {
			continue
		}

		newPeak = math.Max(newPeak, float64(dailyMetric.PlayerCount))
		total += dailyMetric.PlayerCount
		numCounted++
	}

	var newAverage int
	if numCounted == 0 {
		newAverage = 0
	} else {
		newAverage = total / numCounted
	}

	log.Printf("Computed average player count of: %d on month: %d using %d dates.\n",
		newAverage, monthToAvg, numCounted)

	err = db.UpdateDailyMetricList(app.Ref.ID, &newDailyMetricList)
	if err != nil {
		log.Printf("Error updating daily metric list (removal) for app: %s", app.Ref.ID.String())
	}

	var previousMonthMetrics *db.Metric
	previousMonthMetrics, err = db.GetPreviousMonthMetric(app.Ref.ID)
	if err != nil {
		log.Printf("Error retrieving previous month metrics for app %s.\n", app.Ref.ID.String())
		return err
	}

	newMonth := constructNewMonthMetric(previousMonthMetrics, newPeak, float64(newAverage), monthToAvg, yearToAvg)
	err = db.InsertMonthly(app.Ref.ID, newMonth)
	if err != nil {
		log.Printf("Error updating new month metrics for app %s.\n", app.Ref.ID.String())
		return err
	}
	return nil
}

func processApp(app *db.AppRef) error {
	dm, err := stats.Fetch(app.Date, app.Ref.Domain, app.Ref.DomainID)
	if err != nil {
		err = db.InsertException(app)
		if err != nil {
			log.Printf("Error inserting app %d to exception queue! %s\n", app.Ref.DomainID, err)
			// What do?
		}
		return err
	}

	err = db.InsertDaily(app.Ref.ID, dm)
	if err != nil {
		err = db.InsertException(app)
		if err != nil {
			log.Printf("Error inserting app %d to exception queue! %s\n", app.Ref.DomainID, err)
			// What do?
		}
		return err
	}

	return nil
}

// RecoverExceptions : Best effort to retry all exception instances
func RecoverExceptions() {
	var appsToUpdate, err = db.GetExceptions()
	if err != nil {
		log.Printf("Error retrieving exceptions. %s", err)
		return
	}

	for _, app := range *appsToUpdate {
		err = processApp(&app)
		if err != nil {
			log.Printf("Daily retry (%s) failed for app: %+v - %s", app.Date, app.Ref.ID, err)
			continue
		}
	}
}

func constructNewMonthMetric(previous *db.Metric, peak float64, avg float64,
	month time.Month, year int) *db.Metric {

	var gainStr string
	var gainPcStr string
	if previous != nil {
		gain := avg - float64(previous.AvgPlayers)
		gainPc := gain / float64(previous.AvgPlayers)
		gainStr = fmt.Sprintf("%.2f", gain)
		gainPcStr = fmt.Sprintf("%.2f%%", gainPc)
	} else {
		gainStr = "-"
		gainPcStr = "-"
	}

	// Construct new month metric
	var newMonthMetric = db.Metric{
		Date:        time.Date(year, month, 1, 0, 0, 0, 0, time.UTC),
		AvgPlayers:  int(avg),
		Gain:        gainStr,
		GainPercent: gainPcStr,
		Peak:        int(peak),
	}

	return &newMonthMetric
}
