package tracula

import (
	"fmt"
	"sort"
	"time"
)

const (
	HOURSPERDAY    = 24
	RETENTIONLIMIT = 90
)

// Current handles two types of data: Metric and DailyMetric
// Duplicate code still exists but at least it exists as a single function
// Perhaps look into a better way to do this; polymorphism
func sortDates(m interface{}) {
	switch t := m.(type) {
	case []Metric:
		if sort.SliceIsSorted(t, func(i int, j int) bool {
			return t[i].Date.Before(t[j].Date)
		}) {
			return
		}
		sort.Slice(t, func(i int, j int) bool {
			return t[i].Date.Before(t[j].Date)
		})
	case []DailyMetric:
		if sort.SliceIsSorted(t, func(i int, j int) bool {
			return t[i].Date.Before(t[j].Date)
		}) {
			return
		}
		sort.Slice(t, func(i int, j int) bool {
			return t[i].Date.Before(t[j].Date)
		})
	default:
		// Do nothing for now...
	}
}

func monthlySanitise(appBom *App, currentDateTime *time.Time) (*[]DailyMetric, int, int) {
	var newDailyMetricList []DailyMetric

	var total int = 0
	var numCounted int = 0
	var newPeak int = 0

	// Criteria for including metric in the monthly calculation:
	// 1. Before today's date
	// 2. Element month = current month - 1 (normally this process called on the first day of the month)
	targetMonth := currentDateTime.Month() - 1
	if targetMonth == 0 {
		targetMonth = time.December
	}

	// Criteria for purge:
	// 1. day difference <= RETENTIONLIMIT
	for _, dailyMetric := range appBom.DailyMetrics {

		if dayDiff(currentDateTime, &dailyMetric.Date) >= RETENTIONLIMIT {
			continue
		}

		if targetMonth == dailyMetric.Date.Month() {
			newPeak = max(newPeak, dailyMetric.PlayerCount)
			total += dailyMetric.PlayerCount
			numCounted++
		}
		newDailyMetricList = append(newDailyMetricList, dailyMetric)
	}

	sortDates(newDailyMetricList)

	var newAverage int = 0
	if numCounted > 0 {
		newAverage = total / numCounted
	}
	return &newDailyMetricList, newPeak, newAverage
}

// dayDiff calculates the number of days from : a - b
// Assumption that there are 24 hours in a day
func dayDiff(a, b *time.Time) int {
	return int(a.Sub(*b).Hours() / HOURSPERDAY)
}

func constructNewMonthMetric(previous *Metric, peak int, avg int, cdt *time.Time) *Metric {
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

	var targetMonth time.Month = cdt.Month() - 1
	var targetYear int = cdt.Year()
	if cdt.Month() == 0 {
		targetMonth = time.December
		targetYear--
	}

	// Construct new month metric
	var newMonthMetric = Metric{
		Date:        time.Date(targetYear, targetMonth, 1, 0, 0, 0, 0, time.UTC),
		AvgPlayers:  avg,
		Gain:        gainStr,
		GainPercent: gainPcStr,
		Peak:        peak,
	}
	return &newMonthMetric
}
