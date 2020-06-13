package pc

import (
	"cloud.google.com/go/logging"
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"os"
)

type loggers struct {
	Info  *log.Logger
	Debug *log.Logger
	Error *log.Logger
}

// Collections struct containing MongoDB collections to be used
type Collections struct {
	Stats      *mongo.Collection
	Exceptions *mongo.Collection
}

// Config for execution
type Config struct {
	Ctx          context.Context
	Col          *Collections
	Trace        *loggers
	LoggerClient *logging.Client
	LocalEnabled bool
}

// InitConfig - initialise config struct and return pointer to it
func InitConfig(ctx context.Context, cols *Collections) *Config {
	newLoggers, loggerClient := initCloudLoggers(ctx)
	newConfig := Config{
		Ctx:          ctx,
		Col:          cols,
		Trace:        newLoggers,
		LoggerClient: loggerClient,
		LocalEnabled: false,
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
