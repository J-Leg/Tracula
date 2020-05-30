package recovery

import (
	"context"
	"fmt"
	"net/http"
	"playercount/src/core"
	"time"
)

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
