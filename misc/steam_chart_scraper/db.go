package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

// DB Constants
const (
	URI       = "mongodb://localhost:27017"
	DBTIMEOUT = 10
)

// Client public DB client
var client = initConnection()

func commitDbEntry(entry *DbEntry) {
	metricsCollection := client.Database("webapp").Collection("playerCount")

	log.Println(fmt.Sprintf("Storing entry: %d to DB", entry.AppID))
	res, err := metricsCollection.InsertOne(context.Background(), entry)
	if err != nil {
		log.Fatalf(fmt.Sprintf("Error inserting to DB: %s", err))
	}
	log.Println(fmt.Sprintf("result: %s", res))

}

func initConnection() *mongo.Client {
	client, err := mongo.NewClient(options.Client().ApplyURI(URI))
	if err != nil {
		log.Fatalf(fmt.Sprintf("Error initialising client: %s", URI))
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), DBTIMEOUT*time.Second)
	defer cancelFunc()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatalf(fmt.Sprintf("Error connecting to database: %s", err))
	}
	return client
}
