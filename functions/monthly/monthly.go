package monthly

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
