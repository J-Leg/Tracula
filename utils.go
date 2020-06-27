package tracula

import (
	"fmt"
	"sort"
	"time"
)

// Current handles two types of data: Metric and DailyMetric
// Duplicate code still exists but at least it exists as a single function
// Perhaps look into a better way to do this; polymorphism
func sortDates(m interface{}) {
	switch t := m.(type) {
	case *[]Metric:
		list := *t
		if sort.SliceIsSorted(list, func(i int, j int) bool {
			return list[i].Date.Before(list[j].Date)
		}) {
			return
		}
		sort.Slice(list, func(i int, j int) bool {
			return list[i].Date.Before(list[j].Date)
		})
	case *[]DailyMetric:
		list := *t
		if sort.SliceIsSorted(list, func(i int, j int) bool {
			return list[i].Date.Before(list[j].Date)
		}) {
			return
		}
		sort.Slice(list, func(i int, j int) bool {
			return list[i].Date.Before(list[j].Date)
		})
	default:
		// Do nothing for now...
	}
}

func monthlySanitise(appBom *App, monthToAvg time.Month, yearToAvg int) (*[]DailyMetric, int, int) {
	// Initialise a new list to be stored
	var newDailyMetricList []DailyMetric

	var total int = 0
	var numCounted int = 0
	var newPeak int = 0

	for _, dailyMetric := range (*appBom).DailyMetrics {
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

		newPeak = max(newPeak, dailyMetric.PlayerCount)
		total += dailyMetric.PlayerCount
		numCounted++
	}

	sortDates(newDailyMetricList)

	var newAverage int = 0
	if numCounted > 0 {
		newAverage = total / numCounted
	}
	return &newDailyMetricList, newPeak, newAverage
}

func constructNewMonthMetric(previous *Metric, peak int, avg int,
	month time.Month, year int) *Metric {

	var gainStr string = "-"
	var gainPcStr string = "-"
	if previous != nil {
		gain := avg - previous.AvgPlayers
		if previous.AvgPlayers > 0 {
			gainPc := gain / previous.AvgPlayers
			gainPcStr = fmt.Sprintf("%.2f%%", float32(gainPc))
		}
		gainStr = fmt.Sprintf("%d", gain)
	}

	// Construct new month metric
	var newMonthMetric = Metric{
		Date:        time.Date(year, month, 1, 0, 0, 0, 0, time.UTC),
		AvgPlayers:  avg,
		Gain:        gainStr,
		GainPercent: gainPcStr,
		Peak:        peak,
	}

	return &newMonthMetric
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
