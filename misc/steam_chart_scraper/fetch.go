package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Steamcharts constants
const (
	DOMAIN    = "https://steamcharts.com"
	APP       = "/app/"
	BATCHBASE = "/top/p."
	CAP       = 449
	TIMEOUT   = 15
)

func fetchAll() {
	client := &http.Client{
		Timeout: TIMEOUT * time.Second,
	}

	for i := 435; i <= CAP; i++ {
		fetchBatchPage(client, i)
	}
}

func fetchGamePage(id int, gameName string) bool {
	url := DOMAIN + APP + strconv.Itoa(id)
	log.Println(fmt.Sprintf("Fetching stats on %s...", url))

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error: %v", err)
		return false
	}

	document, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal("Error reading page")
		return false
	}

	tableElem := document.Find("tbody").Find("tr")
	length := tableElem.Length()

	// Move attributes to the singleton DbEntry struct
	InitDbEntry(gameName, id, length)

	tableElem.Each(func(i int, elem *goquery.Selection) {
		var columns [METRICSIZE]string
		elem.Find("td").Each(func(idx int, s *goquery.Selection) {
			columns[idx] = strings.TrimSpace(s.Text())
		})
		InsertMetric(i, columns)
	})

	log.Println("Commit DB entry")
	commitDbEntry(getEntry())
	return true
}

func fetchBatchPage(client *http.Client, index int) {
	// Make HTTP request
	log.Println(fmt.Sprintf("Requesting page: %d\n", index))
	response, err := client.Get(fmt.Sprintf("%s%d", DOMAIN+BATCHBASE, index))
	if err != nil {
		log.Fatal(err)
		return
	}
	defer response.Body.Close()

	// Create a goquery document from the HTTP response
	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal("Error loading HTTP response body. ", err)
	}
	document.Find(".game-name").Each(processElement)
}

// This will get called for each HTML element found
func processElement(index int, element *goquery.Selection) {
	linkElement := element.Find("a")
	href, exists := linkElement.Attr("href")
	if !exists {
		log.Println(fmt.Sprintf("Element %d is invalid", index))
		return
	}

	gameNameString := strings.TrimSpace(linkElement.Text())
	var splits = strings.Split(strings.TrimSpace(href), "/")
	var id, err = strconv.Atoi(splits[len(splits)-1])
	if err != nil {
		log.Fatalf(fmt.Sprintf("Error retrieving ID from: %s", strings.TrimSpace(href)))
	}

	log.Println(fmt.Sprintf("Fetching stats for Name: %s - ID: %d", gameNameString, id))
	if !fetchGamePage(id, gameNameString) {
		return
	}
}
