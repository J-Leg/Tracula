// This file is purely for local development
package main

import (
	"log"
	"os"

	// "cloud.google.com/go/logging"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"playercount"
)

func main() {
	funcframework.RegisterHTTPFunction("/", playercount.ProcessMonthly)

	// Use PORT environment variable, or default to 8080.
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}
