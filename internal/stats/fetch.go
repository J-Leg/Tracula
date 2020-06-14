package stats

// Fetch returns a pointer to a DailyMetric struct if retrieval process succeeded,
// otherwise an error is returned
func Fetch(domain string, id int) (int, error) {
	var err error
	var fetchRes int
	if domain == "steam" {
		fetchRes, err = fetchSteam(id)
	} else {
		fetchRes, err = fetchOsrs()
	}
	if err != nil {
		return 0, err
	}
	return fetchRes, nil
}
