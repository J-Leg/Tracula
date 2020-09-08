package tracula

import (
	"github.com/J-Leg/tracula/internal/core"
)


// Execute : Core execution for daily updates
// Update all apps
func ExecuteDaily(cfg *core.Config) {
  core.Daily(cfg)
}

// ExecuteMonthly : Monthly process
func ExecuteMonthly(cfg *core.Config) {
  core.Monthly(cfg)
}

// ExecuteTracker runs a job to aggregate any apps that are worth tracking
func ExecuteTracker(cfg *core.Config) {
  core.Track(cfg)
}

// ExecuteRefresh updates the app library
func ExecuteRefresh(cfg *core.Config) {
  core.Refresh(cfg)
}

// ExecuteRecovery : Best effort to retry all exception instances
func ExecuteRecovery(cfg *core.Config) {
  core.Recover(cfg)
}

