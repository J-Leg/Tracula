package playercount

import (
	"cloud.google.com/go/logging"
	"context"
	"fmt"
	"net/http"
	"playercount/src/core"
	"time"
)

// ProcessMonthly is an HTTP Cloud Function.
func ProcessMonthly(w http.ResponseWriter, r *http.Request) {

	// Create dependencies
	var ctx context.Context = context.Background()
	var logger logging.Logger = initLogger(ctx)

	// First day of the month
	// if time.Now().Day() == 1 {
	var executionElapsed time.Duration
	var start time.Time
	var dStart time.Time

	fmt.Println("~~~~~~~ Execute Monthly Update ~~~~~~~")
	start = time.Now()
	core.ExecuteMonthly()
	dStart = time.Now()
	fmt.Println("~~~~~~~ Monthly Update Complete ~~~~~~~")
	executionElapsed = dStart.Sub(start)

	fmt.Printf("Total elapsed (Monthly) execution time: %s\n\n", executionElapsed.String())
	// }
}
