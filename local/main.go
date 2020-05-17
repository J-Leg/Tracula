package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"playercount/src/core"
	"time"
)

func main() {
	executeMonthly()
	// execute()
	// recover()
}

// func recover() {
// 	f, err := os.OpenFile("logs/recover.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
// 	if err != nil {
// 		log.Fatalf("error opening file: %v", err)
// 	}
// 	log.SetOutput(f)

// 	fmt.Println("~~~~~~~ Recovery ~~~~~~~")
// 	start := time.Now()
// 	core.RecoverExceptions()
// 	dStart := time.Now()
// 	fmt.Println("~~~~~~~ Recovery Complete ~~~~~~~")
// 	f.Close()
// 	recoveryElapsed := dStart.Sub(start)

// 	fmt.Printf("Total elapsed recovery time: %s\n\n", recoveryElapsed.String())
// }

func executeMonthly() {

	f, err := os.OpenFile("../logs/exec_monthly.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)

	// First day of the month
	// if time.Now().Day() == 1 {
	var executionElapsed time.Duration
	var start time.Time
	var dStart time.Time

	fmt.Println("~~~~~~~ Execute Monthly Update ~~~~~~~")
	start = time.Now()
	core.ExecuteMonthly(context.Background())
	dStart = time.Now()
	fmt.Println("~~~~~~~ Monthly Update Complete ~~~~~~~")
	f.Close()
	executionElapsed = dStart.Sub(start)

	fmt.Printf("Total elapsed (Monthly) execution time: %s\n\n", executionElapsed.String())
	// }
}

// func execute() {
// 	var executionElapsed time.Duration
// 	var start time.Time
// 	var dStart time.Time

// 	f, err := os.OpenFile("logs/exec.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
// 	if err != nil {
// 		log.Fatalf("error opening file: %v", err)
// 	}
// 	log.SetOutput(f)

// 	fmt.Println("~~~~~~~ Execute Daily Update ~~~~~~~")
// 	start = time.Now()
// 	core.Execute()
// 	dStart = time.Now()
// 	fmt.Println("~~~~~~~ Daily Update Complete ~~~~~~~")
// 	f.Close()
// 	executionElapsed = dStart.Sub(start)

// 	fmt.Printf("Total elapsed (Daily) execution time: %s\n\n", executionElapsed.String())
// }
