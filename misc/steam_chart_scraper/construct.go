package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// DB Constants
const (
	METRICSIZE  = 5
	DATEPATTERN = "1-January-2006 00:00:00"
)

// Client public DB client
var entry = new(DbEntry)

// DbEntry - Struct for holding DB entry
type DbEntry struct {
	Name         string        `bson:"name"`
	AppID        int           `bson:"app_id"`
	Metrics      []Metric      `bson:"metrics"`
	DailyMetrics []DailyMetric `bson:"daily_metrics"`
	Domain       string        `bson:"domain"`
}

// DailyMetric - Metric obj
type DailyMetric struct {
	Date        time.Time `bson:"date"`
	PlayerCount int       `bson:"player_count"`
}

// Metric element
type Metric struct {
	Date        time.Time `bson:"date"`
	AvgPlayers  int       `bson:"avg_players"`
	Gain        string    `bson:"gain"`
	GainPercent string    `bson:"gain_percent"`
	Peak        int       `bson:"peak_players"`
}

func abs(a int, b int) int {
	if a > b {
		return a - b
	}

	return b - a
}

// InitDbEntry initialise DB entry
func InitDbEntry(name string, id int, length int) {
	log.Println(fmt.Sprintf("LENGTH: %d", length))
	entry.Name = name
	entry.AppID = id
	entry.Domain = "steam"
	entry.Metrics = make([]Metric, length)
	log.Println(fmt.Sprintf("%d", len(entry.Metrics)))
}

// InsertMetric initialise metric and append to entry
func InsertMetric(index int, columns [METRICSIZE]string) {
	date := strings.Split(columns[0], " ")
	if columns[0] == "Last 30 Days" {
		now := time.Now()
		date[0] = now.Month().String()
		date[1] = strconv.Itoa(now.Year())
	}

	s := fmt.Sprintf("10-%s-%s 00:00:00", date[0], date[1])
	dateTime, err := time.Parse(DATEPATTERN, s)
	if err != nil {
		log.Fatalf("Error computing date")
	}

	avgFloat, err := strconv.ParseFloat(columns[1], 32)
	if err != nil {
		log.Println("Error converting float to int")
	}
	avg := int(avgFloat)

	peakFloat, err := strconv.ParseFloat(columns[4], 32)
	if err != nil {
		log.Println("Error converting float to int")
	}
	peak := int(peakFloat)

	newMetric := Metric{Date: dateTime,
		AvgPlayers:  avg,
		Gain:        columns[2],
		GainPercent: columns[3],
		Peak:        peak,
	}

	entry.Metrics[index] = newMetric
}

func getEntry() *DbEntry {
	return entry
}
