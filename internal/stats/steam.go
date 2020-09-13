package stats

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

// constants
const (
	STEAMDOMAIN = "https://api.steampowered.com"
	// Population
	POPULATIONINTERFACE = "ISteamUserStats"
	POPULATIONFUNCTION  = "GetNumberOfCurrentPlayers"
	POPULATIONVERSION   = "v1"
	IDENTIFIER          = "?appid="
	POPULATIONBASE      = STEAMDOMAIN + "/" + POPULATIONINTERFACE + "/" + POPULATIONFUNCTION + "/" + POPULATIONVERSION + "/" + IDENTIFIER
	// Apps
	APPINTERFACE = "ISteamApps"
	APPFUNCTION  = "GetAppList"
	APPVERSION   = "v2"
	APPBASE      = STEAMDOMAIN + "/" + APPINTERFACE + "/" + APPFUNCTION + "/" + APPVERSION
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
	url := POPULATIONBASE + strconv.Itoa(id)

	res := 0
	r, err := myClient.Get(url)
	if err != nil {
		return res, err
	}
	defer r.Body.Close()

	serialResult, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return res, err
	}

	var rc ResponseContainer
	err = json.Unmarshal(serialResult, &rc)
	if err != nil {
		return res, err
	}

	res = rc.Data.Count
	return res, nil
}

type AppResponseContainer struct {
	AppList AppListDataContainer `json:"applist"`
}

type AppListDataContainer struct {
	Apps []AppDataContainer `json:"apps"`
}

type AppDataContainer struct {
	ID   int    `json:"appid"`
	Name string `json:"name"`
}

func fetchSteamApps() (map[int]string, error) {
	var appMap map[int]string = make(map[int]string)
	url := APPBASE

	r, err := myClient.Get(url)
	if err != nil {
		return appMap, err
	}

	serialResult, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return appMap, err
	}

	var responseContainer AppResponseContainer
	err = json.Unmarshal(serialResult, &responseContainer)
	if err != nil {
		return appMap, err
	}

	// Convert response to map
	for _, element := range responseContainer.AppList.Apps {
		appMap[element.ID] = element.Name
	}
	return appMap, nil
}
