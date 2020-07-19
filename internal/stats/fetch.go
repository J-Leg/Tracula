package stats

import (
	"errors"
	"fmt"
	"strings"
)

// Fetch returns a pointer to a DailyMetric struct if retrieval process succeeded,
// otherwise an error is returned
func Fetch(domain string, id int) (int, error) {
	var err error
	var fetchRes int
	if domain == "steam" {
		fetchRes, err = fetchSteam(id)
	} else if domain == "osrs" {
		fetchRes, err = fetchOsrs()
	} else {
		// Unknown domain
		err = errors.New(fmt.Sprintf("Unknown domain: %s when fetching", domain))
	}
	if err != nil {
		return 0, err
	}
	return fetchRes, nil
}

// FetchApps returns a set of appIds
func FetchApps() (map[string]map[int]string, error) {
	var domainAppMap map[string]map[int]string = make(map[string]map[int]string)

	var err error
	var errorStrings []string

	// Add to the map for each domain
	res, err := fetchSteamApps()
	if err != nil {
		errorStrings = append(errorStrings, "Steam: "+err.Error())
	} else {
		domainAppMap["steam"] = res
	}

	if len(errorStrings) == 0 {
		return domainAppMap, nil
	}
	return domainAppMap, fmt.Errorf(strings.Join(errorStrings, "\n"))
}
