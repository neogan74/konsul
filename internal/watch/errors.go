// Package watch represents the watch API
package watch

import "errors"

var (
	// ErrTooManyWatchers is returned when a client exceeds the maximum number of watchers
	ErrTooManyWatchers = errors.New("too many watchers for this client")

	// ErrWatcherNotFound is returned when a watcher ID is not found
	ErrWatcherNotFound = errors.New("watcher not found")

	// ErrInvalidPattern is returned when a watch pattern is invalid
	ErrInvalidPattern = errors.New("invalid watch pattern")
)
