package stats

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

// constants
const (
	STEAMDOMAIN = "https://api.steampowered.com"
	INTERFACE   = "ISteamUserStats"
	FUNCTION    = "GetNumberOfCurrentPlayers"
	VERSION     = "v1"
	IDENTIFIER  = "?appid="
	BASEURL     = STEAMDOMAIN + "/" + INTERFACE + "/" + FUNCTION + "/" + VERSION + "/" + IDENTIFIER
)

var myClient = &http.Client{Timeout: 10 * time.Second}

// ResponseContainer json response from steam API
type ResponseContainer struct {
	Data DataContainer `json:"response"`
}

// DataContainer player container
type DataContainer struct {
	Count  int `json:"player_count"`
	Result int `json:"result"`
}

func fetchSteam(id int) (int, error) {
	url := BASEURL + strconv.Itoa(id)
	log.Println(fmt.Sprintf("[STEAM] Fetching current player count on %s", url))

	res := 0
	r, err := myClient.Get(url)
	if err != nil {
		return res, err
	}
	defer r.Body.Close()

	resultByteArr, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return res, err
	}

	var rc ResponseContainer
	err = json.Unmarshal(resultByteArr, &rc)
	if err != nil {
		return res, err
	}

	res = rc.Data.Count
	log.Println(fmt.Sprintf("Current total player count for url: %d", res))
	return res, nil
}
