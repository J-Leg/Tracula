package daily

import (
	"context"
	"fmt"
	"net/http"
	"playercount/src/core"
	"time"
)

// ProcessDaily - Daily process receptor
func ProcessDaily(w http.ResponseWriter, r *http.Request) {
	var ctx context.Context = context.Background()

	start := time.Now()

	fmt.Println("~~~~~~~ Execute Daily Update ~~~~~~~")
	core.Execute(ctx)
	fmt.Println("~~~~~~~ Daily Update Complete ~~~~~~~")

	end := time.Now()

	executionElapsed := end.Sub(start)
	fmt.Printf("Total elapsed (Daily) execution time: %s\n\n", executionElapsed.String())
}
