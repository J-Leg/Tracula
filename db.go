package tracula

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

// DB Constants
const (
	DBTIMEOUT   = 10
	DATEPATTERN = "2006-01-02 15:04:05"
)

// Config is an object that holds the config info of the application running
// Only necessary fields from gloabal config, need to find a better way to do this...

// App is the data structure for an element in the DB
type App struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Name         string             `bson:"name"`
	AppID        int                `bson:"app_id"`
	Metrics      []Metric           `bson:"metrics"`
	DailyMetrics []DailyMetric      `bson:"daily_metrics"`
	Domain       string             `bson:"domain"`
}

// DailyMetric - Metric obj
type DailyMetric struct {
	Date        time.Time `bson:"date"`
	PlayerCount int       `bson:"player_count"`
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
	GetAppList() ([]AppShadow, error)
	GetMonthlyList(id primitive.ObjectID) (*[]Metric, error)
	GetDailyList(id primitive.ObjectID) (*[]DailyMetric, error)
	GetExceptions() (*[]AppShadow, error)

	PushDaily(id primitive.ObjectID, element *DailyMetric) error
	PushMonthly(id primitive.ObjectID, element *Metric) error
	PushException(app *AppShadow) error

	UpdateApp(id primitive.ObjectID, newMetricList *[]Metric) error
	UpdateDailyList(id primitive.ObjectID, newMetricList *[]DailyMetric) error

	FlushExceptions()
}

// GetAppList : Get List of Apps as AppMeta
func (cfg Config) GetAppList() ([]AppShadow, error) {

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

	localCtx := cfg.Ctx

	// Query options
	// Only want fields corresponding to dbAppRef
	opts := options.Find().SetProjection(proj)
	cursor, err := cfg.Col.Stats.Find(localCtx, match, opts)
	if err != nil {
		cfg.Trace.Error.Printf("cursor construction failed.\n")
		return nil, err
	}

	dateTime, err := time.Parse(DATEPATTERN, time.Now().UTC().String()[:19])
	if err != nil {
		cfg.Trace.Error.Printf("unable to construct datetime.")
		return nil, err
	}

	var appList []AppShadow
	for cursor.Next(localCtx) {
		var dbEntry dbAppRef

		if err := cursor.Decode(&dbEntry); err != nil {
			cfg.Trace.Error.Printf("Error decoding DB entry. %s", err)
			continue
		}

		aNewShadow := AppShadow{
			Ref:  dbEntry,
			Date: dateTime,
		}

		appList = append(appList, aNewShadow)
	}
	cursor.Close(localCtx)
	return appList, nil
}

// PushDaily : Insert daily metric
func (cfg Config) PushDaily(id primitive.ObjectID, element *DailyMetric) error {
	cfg.Trace.Debug.Printf("[PlayerCount Collection] inserting new daily for app %s", id.String())

	match := bson.M{"_id": id}
	action := bson.M{"$push": bson.M{"daily_metrics": element}}
	_, err := cfg.Col.Stats.UpdateOne(cfg.Ctx, match, action)
	if err != nil {
		return err
	}
	cfg.Trace.Debug.Printf("[PlayerCount Collection] insertion success.")
	return nil
}

func (cfg Config) PushApp(element *App) error {
	cfg.Trace.Debug.Printf("[PlayerCount Collection] inserting new app: %s", element.Name)

	_, err := cfg.Col.Stats.InsertOne(cfg.Ctx, element)
	if err != nil {
		return err
	}
	cfg.Trace.Debug.Printf("[PlayerCount Collection] insertion success.")
	return nil
}

// PushMonthly : Insert new month instance
func (cfg Config) PushMonthly(id primitive.ObjectID, element *Metric) error {
	cfg.Trace.Debug.Printf("[PlayerCount Collection] inserting new monthly for app %s.", id.String())

	match := bson.M{"_id": id}
	action := bson.M{"$push": bson.M{"metrics": element}}
	res, err := cfg.Col.Stats.UpdateOne(cfg.Ctx, match, action)
	if err != nil {
		return err
	}
	cfg.Trace.Debug.Printf("[PlayerCount Collection] insertion success - %+v.\n", res)
	return nil
}

// PushException : Insert exception instance
func (cfg Config) PushException(app *AppShadow) error {
	cfg.Trace.Debug.Printf("[Exception Queue] inserting daily update for app %s [%s]: %s \n",
		app.Ref.Name, app.Ref.ID.String(), app.Date.String())

	res, err := cfg.Col.Exceptions.InsertOne(cfg.Ctx, app)
	if err != nil {
		return err
	}
	cfg.Trace.Debug.Printf("[Exception Queue] insertion success - %s.\n", res)
	return nil
}

// UpdateApp : Update app
func (cfg Config) UpdateApp(app *App) error {
	cfg.Trace.Debug.Printf("[PlayerCount Collection] updating App: %s.\n", app.ID.String())

	match := bson.M{"_id": app.ID}
	update := bson.M{
		"metrics":       app.Metrics,
		"daily_metrics": app.DailyMetrics,
	}
	action := bson.M{"$set": update}

	res, err := cfg.Col.Stats.UpdateOne(cfg.Ctx, match, action)
	if err != nil {
		return err
	}
	cfg.Trace.Debug.Printf("[PlayerCount Collection] update success - %+v\n.", res)
	return nil
}

// GetMonthlyList : retrieve previous month's metrics
func (cfg Config) GetMonthlyList(id primitive.ObjectID) (*[]Metric, error) {
	cfg.Trace.Debug.Printf("[PlayerCount Collection] retrieve month metrics for app %s.\n", id.String())

	var app App
	match := bson.M{"_id": id}
	err := cfg.Col.Stats.FindOne(cfg.Ctx, match).Decode(&app)
	if err != nil {
		cfg.Trace.Error.Printf("error retrieving document from DB: %s\n", err)
		if err == mongo.ErrNoDocuments {
			cfg.Trace.Debug.Printf("app id %s does not exist.\n", id.String())
		}
		return nil, err
	}
	cfg.Trace.Debug.Printf("[PlayerCount Collection] retrieval success.\n")

	monthlyMetricList := app.Metrics
	if len(monthlyMetricList) == 0 {
		log.Printf("ID %s has empty monthly metric list", id.String())
		return nil, nil
	}

	return &monthlyMetricList, nil
}

// GetDailyList retrives a list of daily metrics for input app
// UNUSED
func (cfg Config) GetDailyList(id primitive.ObjectID) (*[]DailyMetric, error) {
	cfg.Trace.Debug.Printf("[PlayerCount Collection] retrieving daily metric list for app %s.\n", id.String())
	var result App

	match := bson.M{"_id": id}
	err := cfg.Col.Stats.FindOne(cfg.Ctx, match).Decode(&result)
	if err != nil {
		log.Printf("Error retrieving document from DB: %s\n", err)
		if err == mongo.ErrNoDocuments {
			log.Printf("ID %s does not exist in Stats DB.\n", id.String())
		}
		return nil, err
	}
	cfg.Trace.Debug.Printf("[PlayerCount Collection] retrieval success.\n")
	return &result.DailyMetrics, nil
}

// GetApp retrieves the entire document and returns it as an struct app
func (cfg Config) GetApp(id primitive.ObjectID) (*App, error) {
	cfg.Trace.Debug.Printf("[PlayerCount Collection] retrieving daily metric list for app %s.\n", id.String())

	var result App
	match := bson.M{"_id": id}
	err := cfg.Col.Stats.FindOne(cfg.Ctx, match).Decode(&result)
	if err != nil {
		log.Printf("Error retrieving document from DB: %s\n", err)
		if err == mongo.ErrNoDocuments {
			log.Printf("ID %s does not exist in Stats DB.\n", id.String())
		}
		return nil, err
	}
	cfg.Trace.Debug.Printf("[PlayerCount Collection] retrieval success.\n")
	return &result, nil
}

// UpdateDailyList : Update daily metric list for app
func (cfg Config) UpdateDailyList(id primitive.ObjectID, newMetricList *[]DailyMetric) error {
	cfg.Trace.Debug.Printf("[PlayerCount Collection] updating daily metric list for app %s.\n", id.String())
	var updatedDoc App

	match := bson.M{"_id": id}
	action := bson.M{"$set": bson.M{"daily_metrics": *newMetricList}}

	err := cfg.Col.Stats.FindOneAndUpdate(cfg.Ctx, match, action).Decode(&updatedDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("ID %s does not exist in Stats DB.\n", id.String())
		}
		return err
	}

	cfg.Trace.Debug.Printf("[PlayerCount Collection] update success.\n")
	return nil
}

// GetExceptions - Return list of AppShadows
func (cfg Config) GetExceptions() (*[]AppShadow, error) {
	cfg.Trace.Debug.Printf("[Exception Queue] retrieving all Exceptions\n")
	var appList []AppShadow
	ctx := cfg.Ctx
	cursor, err := cfg.Col.Exceptions.Find(ctx, bson.M{})
	if err != nil {
		cfg.Trace.Error.Printf("cursor construction failed. %s\n", err)
		return nil, err
	}

	if err = cursor.All(ctx, &appList); err != nil {
		cfg.Trace.Error.Printf("cursor Decoding failed. %s\n", err)
		cursor.Close(ctx)
		return nil, err
	}

	cursor.Close(ctx)
	return &appList, nil
}

// FlushExceptions : Clear exception queue
func (cfg Config) FlushExceptions() {
	cfg.Trace.Debug.Printf("[Exception Queue] flushing Exceptions\n")
	res, err := cfg.Col.Exceptions.DeleteMany(cfg.Ctx, bson.M{})
	if err != nil {
		cfg.Trace.Error.Printf("error flushing Exceptions: %s.\n", err)
		return
	}

	cfg.Trace.Debug.Printf("flush success: %+v.\n", res)
	return
}
