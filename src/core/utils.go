package core

import (
	"fmt"
	"github.com/J-Leg/player-count/src/db"
	"sort"
	"time"
)

func monthSort(listPtr *[]db.Metric) {
	list := *listPtr
	if sort.SliceIsSorted(list, func(i int, j int) bool {
		return list[i].Date.Before(list[j].Date)
	}) {
		return
	}

	sort.Slice(list, func(i int, j int) bool {
		return list[i].Date.Before(list[j].Date)
	})
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
