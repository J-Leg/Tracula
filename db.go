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

type App struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Metrics      []Metric           `bson:"metrics"`
	DailyMetrics []DailyMetric      `bson:"daily_metrics"`
	StaticData   StaticAppData      `bson:"static_data"`
}

type AppRef struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	RefID      primitive.ObjectID `bson:"ref_id"`
	StaticData StaticAppData      `bson:"static_data"`
}

type StaticAppData struct {
	Name   string `bson:"name"`
	AppID  int    `bson:"app_id"`
	Domain string `bson:"domain"`
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

// Db interface
type Db interface {
	getAppListMonthData() ([]App, error)
	getAppListBasic() ([]App, error)
	getAppListTracked() ([]AppRef, error)
	getAppListFull() ([]App, error)
	getAppList(projection bson.M) ([]App, error)

	GetExceptions() ([]AppRef, error)

	PushApp(element *App) error
	PushDaily(id primitive.ObjectID, element *DailyMetric) error
	PushMonthly(id primitive.ObjectID, element *Metric) error
	PushException(app *AppRef, currentDateTime *time.Time) error
	pushTrackedApp(app *App) error

	UpdateApp(app *App) error
	UpdateDailyList(id primitive.ObjectID, newMetricList *[]DailyMetric) error

	FlushExceptions()
	flushTrackPool() error
}

func (cfg Config) getAppListBasic() ([]App, error) {
	// Get all fields except for daily metrics
	proj := bson.M{"daily_metrics": 0, "metrics": 0}

	return cfg.getAppList(proj)
}

func (cfg Config) getAppListMonthData() ([]App, error) {
	// Get all fields except for daily metrics
	proj := bson.M{"daily_metrics": 0}

	return cfg.getAppList(proj)
}

func (cfg Config) getAppListFull() ([]App, error) {
	// Get all fields except for daily metrics
	proj := bson.M{}

	return cfg.getAppList(proj)
}

// getAppList : Get List of Apps
func (cfg Config) getAppList(projection bson.M) ([]App, error) {

	// Empty match - searching for all elements
	match := bson.M{}

	var cursor *mongo.Cursor

	localCtx := cfg.Ctx

	// Query options
	// Only want fields corresponding to dbAppRef
	opts := options.Find().SetProjection(projection)
	cursor, err := cfg.Col.Stats.Find(localCtx, match, opts)
	if err != nil {
		cfg.Trace.Error.Printf("cursor construction failed.\n")
		return nil, err
	}

	var appList []App
	for cursor.Next(localCtx) {
		var appResult App

		if err := cursor.Decode(&appResult); err != nil {
			cfg.Trace.Error.Printf("Error decoding DB entry. %s", err)
			continue
		}

		appList = append(appList, appResult)
	}
	cursor.Close(localCtx)
	return appList, nil
}

func (cfg Config) getAppListTracked() ([]AppRef, error) {
	cursor, err := cfg.Col.TrackPool.Find(cfg.Ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	var resultTrackList []AppRef
	for cursor.Next(cfg.Ctx) {
		var elem AppRef
		if err := cursor.Decode(&elem); err != nil {
			continue
		}
		resultTrackList = append(resultTrackList, elem)
	}
	cursor.Close(cfg.Ctx)
	return resultTrackList, nil
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
	cfg.Trace.Debug.Printf("[PlayerCount Collection] inserting new app: %s", element.StaticData.Name)

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
	cfg.Trace.Debug.Printf("[PlayerCount Collection] insertion success - %+v.", res)
	return nil
}

// PushException : Insert exception instance
func (cfg Config) PushException(app *AppRef, date *time.Time) error {
	cfg.Trace.Debug.Printf("[Exception Queue] inserting daily update for app %s [%s]: %s.",
		app.StaticData.Name, app.RefID.String(), date.String())

	res, err := cfg.Col.Exceptions.InsertOne(cfg.Ctx, app)
	if err != nil {
		return err
	}
	cfg.Trace.Debug.Printf("[Exception Queue] insertion success - %s.", res)
	return nil
}

func (cfg Config) pushTrackedApp(app *App) error {
	newElem := AppRef{
		RefID:      app.ID,
		StaticData: app.StaticData,
	}

	_, err := cfg.Col.TrackPool.InsertOne(cfg.Ctx, newElem)
	if err != nil {
		return err
	}

	cfg.Trace.Info.Printf("Pushed app: %s to track pool.", app.StaticData.Name)
	return nil
}

func (cfg Config) flushTrackPool() error {
	_, err := cfg.Col.TrackPool.DeleteMany(cfg.Ctx, bson.M{})
	if err != nil {
		return err
	}
	return nil
}

// UpdateApp : Update app
func (cfg Config) UpdateApp(app *App) error {
	cfg.Trace.Debug.Printf("[PlayerCount Collection] updating App: %s.", app.ID.String())

	match := bson.M{"_id": app.ID}
	var replacedDoc bson.M
	err := cfg.Col.Stats.FindOneAndReplace(cfg.Ctx, match, app).Decode(&replacedDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			cfg.Trace.Debug.Printf("No app exists with app id: %s", app.ID)
			return err
		}
	}
	return nil
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

// GetExceptions - Return list of App
func (cfg Config) GetExceptions() ([]AppRef, error) {
	cfg.Trace.Debug.Printf("[Exception Queue] retrieving all Exceptions\n")
	var appList []AppRef
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
	return appList, nil
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
