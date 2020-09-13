package tracula 

import (
	"github.com/J-leg/tracula/internal/core"
  "github.com/J-leg/tracula/config"
)


// Execute : Core execution for daily updates
// Update all apps
func ExecuteDaily(cfg *config.Config) {
  core.Daily(cfg)
}

// ExecuteMonthly : Monthly process
func ExecuteMonthly(cfg *config.Config) {
  core.Monthly(cfg)
}

// ExecuteTracker runs a job to aggregate any apps that are worth tracking
func ExecuteTracker(cfg *config.Config) {
  core.Track(cfg)
}

// ExecuteRefresh updates the app library
func ExecuteRefresh(cfg *config.Config) {
  core.Refresh(cfg)
}

// ExecuteRecovery : Best effort to retry all exception instances
func ExecuteRecovery(cfg *config.Config) {
  core.Recover(cfg)
}

