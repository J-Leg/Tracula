package playercount

import (
	"context"
	"fmt"
	"net/http"
	"playercount/src/core"
	"time"
)

// ProcessMonthly - Monthly process receptor
func ProcessMonthly(w http.ResponseWriter, r *http.Request) {
	var ctx context.Context = context.Background()

	start := time.Now()

	fmt.Println("~~~~~~~ Execute Monthly Update ~~~~~~~")
	core.ExecuteMonthly(ctx)
	fmt.Println("~~~~~~~ Monthly Update Complete ~~~~~~~")

	end := time.Now()

	executionElapsed := end.Sub(start)
	fmt.Printf("Total elapsed (Monthly) execution time: %s\n\n", executionElapsed.String())
}

// ProcessDaily - Daily process receptor
func ProcessDaily(w http.ResponseWriter, r *http.Request) {
	var ctx context.Context = context.Background()

	start := time.Now()

	fmt.Println("~~~~~~~ Execute Daily Update ~~~~~~~")
	core.ExecuteMonthly(ctx)
	fmt.Println("~~~~~~~ Daily Update Complete ~~~~~~~")

	end := time.Now()

	executionElapsed := end.Sub(start)
	fmt.Printf("Total elapsed (Daily) execution time: %s\n\n", executionElapsed.String())
}

// Recover - Recovery process receptor
func Recover(w http.ResponseWriter, r *http.Request) {
	var ctx context.Context = context.Background()

	start := time.Now()

	fmt.Println("~~~~~~~ Execute Recovery ~~~~~~~")
	core.ExecuteRecovery(ctx)
	fmt.Println("~~~~~~~ Daily Update Complete ~~~~~~~")

	end := time.Now()

	executionElapsed := end.Sub(start)
	fmt.Printf("Total elapsed recovery execution time: %s\n\n", executionElapsed.String())
}
