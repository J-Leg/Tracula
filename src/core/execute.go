package core

import (
	"github.com/cheggaaa/pb/v3"
	"log"
	"os"
	"playercount/src/db"
	"playercount/src/stats"
	"time"
)

// Execute : Core execution for daily updates
// Update all apps
func Execute() {
	var appList = db.GetAppList()

	bar := pb.StartNew(len(appList))
	bar.SetRefreshRate(time.Second)
	bar.SetWriter(os.Stdout)
	bar.Start()

	for _, app := range appList {

		// Update progress
		bar.Increment()
		time.Sleep(time.Millisecond)
		err := processApp(&app)
		if err != nil {
			continue
		}
	}
	bar.Finish()
}

func processApp(app *db.AppRef) error {
	dm, err := stats.Fetch(app.Date, app.Ref.Domain, app.Ref.DomainID)
	if err != nil {
		err = db.InsertException(app)
		if err != nil {
			log.Printf("Error inserting app %d to exception queue! %s\n", app.Ref.DomainID, err)
			// What do?
		}
		return err
	}

	err = db.InsertDaily(dm, app)
	if err != nil {
		err = db.InsertException(app)
		if err != nil {
			log.Printf("Error inserting app %d to exception queue! %s\n", app.Ref.DomainID, err)
			// What do?
		}
		return err
	}

	return nil
}

// RecoverExceptions : Best effort to retry all exception instances
func RecoverExceptions() {
	var appsToUpdate, err = db.GetExceptions()
	if err != nil {
		log.Printf("Error retrieving exceptions. %s", err)
		return
	}

	for _, app := range *appsToUpdate {
		err = processApp(&app)
		if err != nil {
			log.Printf("Daily retry (%s) failed for app: %+v - %s", app.Date, app.Ref.ID, err)
			continue
		}
	}
}
