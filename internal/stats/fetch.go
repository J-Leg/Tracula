package stats

import (
  "fmt"
  "strings"
  "errors"
)

// Fetch returns a pointer to a DailyMetric struct if retrieval process succeeded,
// otherwise an error is returned
func Fetch(domain string, id int) (int, error) {
  var err error
  var res int

  switch domain {
    case "steam":
      res, err = fetchSteam(id)
    case "osrs":
      res, err = fetchOsrs()
    default:
      err = errors.New(fmt.Sprintf("Unknown domain: %s", domain))
  }

  if err != nil { return -1, err }
  return res, nil
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
