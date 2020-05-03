package env

import (
	"cloud.google.com/go/logging"
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

// Config for execution
type Config struct {
	ctx context.Context
	db  *mongo.Database
	log *logging.Logger
}

// InitConfig - initialise config struct
func InitConfig(ctx context.Context) *Config {
	newLogger := initLogger(ctx)
	newDb := initDb(ctx)

	newConfig := Config{
		ctx: ctx,
		db:  newDb,
		log: newLogger,
	}

	return &newConfig
}

func initLogger(ctx context.Context) *logging.Logger {
	projectID := GoDotEnvVariable("PROJ_ID")

	// Creates a client.
	client, err := logging.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Sets the name of the log to write to.
	logName := "my-log"
	logger := client.Logger(logName)

	// Logs "hello world", log entry is visible at
	// Stackdriver Logs.
	return logger
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

	return newDb
}
