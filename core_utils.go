package tracula

import (
	"github.com/cheggaaa/pb/v3"
)

func finalise(t int, numSuccess, numError *int, ch chan<- bool, bar *pb.ProgressBar, cfg *Config) {
	close(ch)

	if bar != nil {
		bar.Finish()
	}

	var jobType string
	if t == DAILY {
		jobType = "daily"
	} else if t == MONTHLY {
		jobType = "monthly"
	} else if t == RECOVERY {
		jobType = "recovery"
	} else if t == REFRESH {
		jobType = "refresh"
	} else if t == TRACKER {
		jobType = "tracker"
	} else {
		cfg.Trace.Error.Printf("Invalid job type %d", t)
	}

	cfg.Trace.Info.Printf("%s execution REPORT:\n    success: %d\n    errors: %d", jobType, *numSuccess, *numError)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
