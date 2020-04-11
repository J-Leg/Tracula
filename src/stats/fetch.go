package stats

import (
	"time"
)

// DailyMetric - Metric obj
type DailyMetric struct {
	Date        time.Time `bson:"date"`
	PlayerCount int       `bson:"player_count"`
}

// Fetch : fetch master
func Fetch(dateTime time.Time, domain string, id int) (*DailyMetric, error) {
	var err error
	var fetchRes int
	if domain == "steam" {
		fetchRes, err = fetchSteam(id)
	} else {
		fetchRes, err = fetchOsrs()
	}
	if err != nil {
		return nil, err
	}

	newDailyMetric := DailyMetric{
		Date:        dateTime,
		PlayerCount: fetchRes,
	}

	return &newDailyMetric, nil
}
