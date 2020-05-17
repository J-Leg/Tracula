package env

import (
	"cloud.google.com/go/logging"
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
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
func InitConfig(ctx context.Context) *Config {
	newDb := initDb(ctx)

	newLoggers, loggerClient := initLoggers(ctx)
	newConfig := Config{
		Ctx:          ctx,
		Db:           newDb,
		Trace:        newLoggers,
		LoggerClient: loggerClient,
	}

	return &newConfig
}

func initLoggers(ctx context.Context) (*loggers, *logging.Client) {
	projectID := GoDotEnvVariable("PROJ_ID")

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

func initDb(ctx context.Context) *mongo.Database {
	var newDb *mongo.Database
	var clientOptions *options.ClientOptions
	var dbURI string
	if GoDotEnvVariable("NODE_ENV") == "prd" {
		dbURI = GoDotEnvVariable("PRD_URI")
		clientOptions = options.Client().ApplyURI(GoDotEnvVariable("PRD_URI"))
	} else {
		dbURI = GoDotEnvVariable("DEV_URI")
		clientOptions = options.Client().ApplyURI(GoDotEnvVariable("DEV_URI"))
	}

	newClient, err := mongo.NewClient(clientOptions)
	if err != nil {
		log.Fatalf("[CRITICAL] Error initialising client. URI: %s", dbURI)
	}

	// To be removed when another DB URI is used
	if GoDotEnvVariable("NODE_ENV") == "prd" {
		newDb = newClient.Database("games_stats_app")
		log.Printf("Target: PRD DB\n")
	} else {
		newDb = newClient.Database("games_stats_app_TST")
		log.Printf("Target: TST DB\n")
	}

	err = newClient.Connect(ctx)
	if err != nil {
		log.Fatalf("[CRITICAL] error connecting client. %s", err)
	}

	return newDb
}
