package env

import (
	"cloud.google.com/go/logging"
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"os"
)

// Constants
const (
	DBTIMEOUT = 5
)

type loggers struct {
	Info  *log.Logger
	Debug *log.Logger
	Error *log.Logger
}

// Config for execution
type Config struct {
	Ctx          context.Context
	Db           *mongo.Database
	Trace        *loggers
	LoggerClient *logging.Client
}

// InitConfig - initialise config struct
func InitConfig(ctx context.Context, db *mongo.Database) *Config {
	newDb := db

	newLoggers, loggerClient := initCloudLoggers(ctx)
	newConfig := Config{
		Ctx:          ctx,
		Db:           newDb,
		Trace:        newLoggers,
		LoggerClient: loggerClient,
	}

	return &newConfig
}

func initCloudLoggers(ctx context.Context) (*loggers, *logging.Client) {
	projectID := os.Getenv("PROJ_ID")

	// Creates a client.
	client, err := logging.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	logName := "player-count"
	logger := client.Logger(logName)

	newLoggers := loggers{
		Info:  logger.StandardLogger(logging.Info),
		Debug: logger.StandardLogger(logging.Debug),
		Error: logger.StandardLogger(logging.Error),
	}

	return &newLoggers, client
}
