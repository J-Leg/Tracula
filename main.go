package main

import (
	"fmt"
	"log"
	"os"
	"playercount/src/core"
	"time"
)

func main() {
	execute()
	recover()
}

func recover() {
	f, err := os.OpenFile("logs/recover.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)

	fmt.Println("~~~~~~~ Recovery ~~~~~~~")
	start := time.Now()
	core.RecoverExceptions()
	dStart := time.Now()
	fmt.Println("~~~~~~~ Recovery Complete ~~~~~~~")
	f.Close()
	recoveryElapsed := dStart.Sub(start)

	fmt.Printf("Total elapsed recovery time: %s\n\n", recoveryElapsed.String())
}

func execute() {
	f, err := os.OpenFile("logs/exec.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)

	fmt.Println("~~~~~~~ Execute Daily Update ~~~~~~~")
	start := time.Now()
	core.Execute()
	dStart := time.Now()
	fmt.Println("~~~~~~~ Daily Update Complete ~~~~~~~")
	f.Close()
	executionElapsed := dStart.Sub(start)

	fmt.Printf("Total elapsed execution time: %s\n\n", executionElapsed.String())
}
