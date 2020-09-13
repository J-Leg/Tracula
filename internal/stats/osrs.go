package stats 

import (
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Steamcharts constants
const (
	DOMAIN  = "https://oldschool.runescape.com/"
	TIMEOUT = 15
)

func fetchOsrs() (int, error) {
	client := &http.Client{
		Timeout: TIMEOUT * time.Second,
	}

	// log.Println("Fetch player count for Oldschool Runescape")
	res := 0
	resp, err := client.Get(DOMAIN)
	if err != nil {
		return res, err
	}

	document, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return res, err
	}

	elem := document.Find(".player-count")
	words := strings.Fields(elem.Text())
	playerCountStr := strings.ReplaceAll(words[3], ",", "")

	res, err = strconv.Atoi(playerCountStr)
	if err != nil {
		return res, err
	}

	// log.Printf("Current total player count: %d", res)

	return res, nil
}
