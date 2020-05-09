package db

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"playercount/src/env"
	"playercount/src/stats" // Way to clear this?
	"sort"
	"time"
)

// DB Constants
const (
	DBTIMEOUT   = 10
	DATEPATTERN = "2006-01-02 15:04:05"
)

// App - Entry in the DB is of this format
type App struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty"`
	Name         string              `bson:"name"`
	AppID        int                 `bson:"app_id"`
	Metrics      []Metric            `bson:"metrics"`
	DailyMetrics []stats.DailyMetric `bson:"daily_metrics"`
	Domain       string              `bson:"domain"`
}

// Metric element
type Metric struct {
	Date        time.Time `bson:"date"`
	AvgPlayers  int       `bson:"avgplayers"`
	Gain        string    `bson:"gain"`
	GainPercent string    `bson:"gainpercent"`
	Peak        int       `bson:"peak"`
}

type dbAppProjection struct {
	ID       int `bson:"_id"`
	Name     int `bson:"name"`
	Domain   int `bson:"domain"`
	DomainID int `bson:"app_id"`
}

type dbAppRef struct {
	ID       primitive.ObjectID `bson:"_id"`
	Name     string             `bson:"name"`
	Domain   string             `bson:"domain"`
	DomainID int                `bson:"app_id"`
}

// AppShadow : app data (no historical data)
type AppShadow struct {
	Date time.Time `bson:"date"`
	Ref  dbAppRef  `bson:"reference"`
}

// Db interface
type Db interface {
	GetAppList() []AppShadow

	PushDaily(id primitive.ObjectID, element *stats.DailyMetric) error
	PushMonthly(id primitive.ObjectID, element *Metric) error
	PushException(app *AppShadow) error
}

// GetAppList : Get List of Apps as AppMeta
func GetAppList() []AppShadow {

	// Empty match - searching for all elements
	match := bson.M{}

	var cursor *mongo.Cursor

	// Define projection
	proj := dbAppProjection{
		ID:       1,
		Name:     1,
		Domain:   1,
		DomainID: 1,
	}

	// Query options
	// Only want fields corresponding to dbAppRef
	opts := options.Find().SetProjection(proj)
	cursor, err := cols.stats.Find(param.ctx, match, opts)
	if err != nil {
		log.Fatal(err)
	}

	dateTime, err := time.Parse(DATEPATTERN, time.Now().UTC().String()[:19])
	if err != nil {
		log.Fatal(err)
	}

	var appList []AppShadow

	for cursor.Next(param.ctx) {
		var dbEntry dbAppRef

		if err := cursor.Decode(&dbEntry); err != nil {
			log.Printf("Error decoding DB entry. %s", err)
			continue
		}

		aNewMetaElement := AppShadow{
			Ref:  dbEntry,
			Date: dateTime,
		}

		appList = append(appList, aNewMetaElement)
	}
	cursor.Close(param.ctx)

	return appList
}

// PushDaily : Insert daily metric
func PushDaily(id primitive.ObjectID, element *stats.DailyMetric) error {
	log.Printf("[PlayerCount Collection] inserting new daily for app %s", id.String())

	match := bson.M{"_id": id}
	action := bson.M{"$push": bson.M{"daily_metrics": newDaily}}
	_, err := cols.stats.UpdateOne(param.ctx, match, action)
	if err != nil {
		return err
	}
	log.Println("[PlayerCount Collection] daily insertion success.")
	return nil
}

// PushMonthly : Insert new month instance
func PushMonthly(id primitive.ObjectID, element *Metric) error {
	log.Printf("[PlayerCount Collection] inserting new monthly for app %s.\n", id.String())

	match := bson.M{"_id": id}
	action := bson.M{"$push": bson.M{"metrics": newMonthly}}
	_, err := cols.stats.UpdateOne(param.ctx, match, action)
	if err != nil {
		return err
	}
	log.Println("[PlayerCount Collection] monthly insertion success.")
	return nil
}

// PushException : Insert exception instance
func PushException(app *AppShadow) error {
	log.Printf("[Exception Queue] inserting daily update for app %s [%s]: %s \n",
		app.Ref.Name, app.Ref.ID.String(), app.Date.String())

	res, err := cols.exceptions.InsertOne(param.ctx, app)
	if err != nil {
		return err
	}
	log.Printf("Added to exception queue %s", res)
	return nil
}

// UpdateMonthlyMetricList : update list
func UpdateMonthlyMetricList(id primitive.ObjectID, newMetricList *[]Metric) error {
	log.Printf("[PlayerCount Collection] inserting new monthly metric list for app %s.\n", id.String())

	match := bson.M{"_id": id}
	action := bson.M{"$set": bson.M{"metrics": *newMetricList}}
	_, err := cols.stats.UpdateOne(param.ctx, match, action)
	if err != nil {
		return err
	}
	log.Println("[PlayerCount Collection] monthly set success.")
	return nil
}

// GetMonthMetricList : retrieve previous month's metrics
func GetMonthMetricList(id primitive.ObjectID) (*[]Metric, error) {
	log.Printf("[PlayerCount Collection] retrieving last month metrics for app %s.\n", id.String())

	var app App
	match := bson.M{"_id": id}
	err := cols.stats.FindOne(param.ctx, match).Decode(&app)
	if err != nil {
		log.Printf("Error retrieving document from DB: %s\n", err)
		if err == mongo.ErrNoDocuments {
			log.Printf("ID %s does not exist in stats DB.\n", id.String())
		}
		return nil, err
	}

	monthlyMetricList := app.Metrics
	if len(monthlyMetricList) == 0 {
		log.Printf("ID %s has empty monthly metric list", id.String())
		return nil, nil
	}

	monthSort(&monthlyMetricList)
	return &monthlyMetricList, nil
}

// GetDailyMetricList : Fetch all daily metrics
func GetDailyMetricList(id primitive.ObjectID) (*[]stats.DailyMetric, error) {
	log.Printf("Retrieving daily metric list for app %s.\n", id.String())
	var result App

	match := bson.M{"_id": id}
	err := cols.stats.FindOne(param.ctx, match).Decode(&result)
	if err != nil {
		log.Printf("Error retrieving document from DB: %s\n", err)
		if err == mongo.ErrNoDocuments {
			log.Printf("ID %s does not exist in stats DB.\n", id.String())
		}
		return nil, err
	}

	return &result.DailyMetrics, nil
}

// UpdateDailyMetricList : Update daily metric list for app
func UpdateDailyMetricList(id primitive.ObjectID, newMetricList *[]stats.DailyMetric) error {
	log.Printf("Retrieving daily metric list for app %s.\n", id.String())
	var updatedDoc App

	match := bson.M{"_id": id}
	action := bson.M{"$set": bson.M{"daily_metrics": *newMetricList}}

	err := cols.stats.FindOneAndUpdate(param.ctx, match, action).Decode(&updatedDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("ID %s does not exist in stats DB.\n", id.String())
		}
		return err
	}

	log.Printf("Successfully updated daily metric list for app ID %s.\n", id.String())
	return nil
}

// GetExceptions - Return list of AppRefs and clear collection
func GetExceptions() (*[]AppShadow, error) {
	var appRefs []AppShadow

	cursor, err := cols.exceptions.Find(param.ctx, bson.M{})
	if err != nil {
		log.Printf("Error processing exceptions. %s", err)
		return nil, err
	}

	if err = cursor.All(param.ctx, &appRefs); err != nil {
		log.Printf("Error assembling excpetions. %s", err)
		cursor.Close(param.ctx)
		return nil, err
	}

	cursor.Close(param.ctx)
	return &appRefs, nil
}

// FlushExceptions :
func FlushExceptions() {
	res, err := cols.exceptions.DeleteMany(param.ctx, bson.M{})
	if err != nil {
		log.Printf("Error flushing exceptions: %s.\n", err)
		return
	}

	log.Printf("Exceptions successfully flushed. %+v\n", res)
	return
}

func monthSort(listPtr *[]Metric) {
	list := *listPtr
	if sort.SliceIsSorted(list, func(i int, j int) bool {
		return list[i].Date.Before(list[j].Date)
	}) {
		return
	}

	log.Println("Unsorted monthly metrics. Sort it!")

	sort.Slice(list, func(i int, j int) bool {
		return list[i].Date.Before(list[j].Date)
	})
}
