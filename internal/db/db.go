package db 

import (
  "context"
  "errors"
  "go.mongodb.org/mongo-driver/bson"
  "go.mongodb.org/mongo-driver/bson/primitive"
  "go.mongodb.org/mongo-driver/mongo"
  "time"
  "github.com/j-leg/tracula/config"
)

// DB Constants
const (
  DBTIMEOUT   = 10
  CAPACITY    = 200000

  DAILY    = 0
  MONTHLY  = 1
  RECOVERY = 2
  REFRESH  = 3
  TRACK    = 4
)

type App struct {
  ID           primitive.ObjectID `bson:"_id,omitempty"`
  Metrics      []Metric           `bson:"metrics"`
  DailyMetrics []DailyMetric      `bson:"daily_metrics"`
  StaticData   StaticAppData      `bson:"static_data"`
  Tracked      bool               `bson:"tracked"`
  LastMetric   DailyMetric        `bson:"last_metric"`
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

func GetJobParams(cfg *config.Config, jobType int) (int, *mongo.Cursor, error) {
  var filter bson.M
  var col *mongo.Collection

  switch jobType {
  case MONTHLY, REFRESH, TRACK:
    filter = bson.M{}
    col = cfg.Col.Stats
  case RECOVERY:
    filter = bson.M{}
    col = cfg.Col.Exceptions
  case DAILY:
    filter = bson.M{"tracked": true}
    col = cfg.Col.Stats
  default:
    return 0, nil, errors.New("Invalid job") 
  }

  var res int64
  var err error

  res, err = col.CountDocuments(cfg.Ctx, filter)
  if err != nil { return 0, nil, err }
  if res > CAPACITY { return 0, nil, errors.New("Over capacity") }

  cursor, err := col.Find(cfg.Ctx, filter) 
  if err != nil { return 0, nil, err }

  return int(res), cursor, nil
}

func AddNewApp(ctx context.Context, element *App, col *mongo.Collection) error {
  _, err := col.InsertOne(ctx, element)
  return err
}

func UpdateApp(ctx context.Context, app *App, col *mongo.Collection) error {
	match := bson.M{"_id": app.ID}
	var replacedDoc bson.M
	return col.FindOneAndReplace(ctx, match, app).Decode(&replacedDoc)
}

func SetTrackFlag(ctx context.Context, id primitive.ObjectID, val bool, col *mongo.Collection) error {
  filter := bson.M{"_id": id}
  update := bson.M{"$set": bson.M{"tracked": val}}

  var upDoc bson.M
  return col.FindOneAndUpdate(ctx, filter, update).Decode(&upDoc)
}

func Flush(ctx context.Context, col *mongo.Collection) error {
  _, err := col.DeleteMany(ctx, bson.M{})
  return err
}

func GetFullStaticData(ctx context.Context, col *mongo.Collection) ([]StaticAppData, error) {
  var match bson.M = bson.M{}
  var resultList []StaticAppData

  numDocs, err := col.CountDocuments(ctx, match)
  if err != nil { return resultList, err }

  cursor, err := col.Find(ctx, match)
  if err != nil { return resultList, err }

  resultList = make([]StaticAppData, numDocs)
  idx := -1 
  for cursor.Next(ctx) {
    var appResult App
    idx++

    err := cursor.Decode(&appResult)
    if err != nil { /* TODO: handle?*/ continue }

    resultList[idx] = appResult.StaticData 
  }
  cursor.Close(ctx)
  return resultList, nil
}
