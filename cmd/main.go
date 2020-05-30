package main

import (
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/joho/godotenv"
	"log"
	"os"
	"playercount/functions/daily"
	"playercount/functions/monthly"
	"playercount/functions/recovery"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env override file.")
	}
}

func main() {
	funcframework.RegisterHTTPFunction("/monthly", monthly.ProcessMonthly)
	funcframework.RegisterHTTPFunction("/daily", daily.ProcessDaily)
	funcframework.RegisterHTTPFunction("/recover", recovery.Recover)

	// Use PORT environment variable, or default to 8080.
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}
