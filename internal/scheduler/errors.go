// Package scheduler provides scheduling functionality for message processing.
package scheduler

import "errors"

var (
	ErrSchedulerAlreadyRunning = errors.New("scheduler is already running")
	ErrSchedulerNotRunning     = errors.New("scheduler is not running")
)
