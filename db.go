package tracula 

import (
  "context"
  "errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

// DB Constants
const (
	DBTIMEOUT   = 10
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

func GetJobParams(jobType int, col *mongo.Collection) (int, *mongo.Cursor, error) {
  var filter bson.M

  switch jobType {
  case 1:
    filter = bson.M{}
  case 2:
    filter = bson.M{}
  case 0:
    filter = bson.M{"tracked": true}
  default:
    return 0, nil, errors.New("Invalid job") 
  }

  var res int64
  var err error

  res, err = col.CountDocuments(context.TODO(), filter)
  if err != nil {
    return 0, nil, err
  }

  if res > 1000000 {
    return 0, nil, errors.New("Over capacity") 
  }

  cursor, err := col.Find(context.TODO(), filter) 
  if err != nil {
    return 0, nil, err
  }

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

func SetTrackFlag(id primitive.ObjectID, val bool, col *mongo.Collection) error {
  filter := bson.M{"_id": id}
  update := bson.M{"$set": bson.M{"tracked": val}}

  var upDoc bson.M
  return col.FindOneAndUpdate(context.TODO(), filter, update).Decode(&upDoc)
}

func Flush(col *mongo.Collection) error {
  _, err := col.DeleteMany(context.TODO(), bson.M{})
  return err
}

func GetFullStaticData(col *mongo.Collection) ([]StaticAppData, error) {
  var match bson.M = bson.M{}
  var resultList []StaticAppData

  ctx := context.TODO()

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
