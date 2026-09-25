package main

import "fmt"

// IPCheckError wraps a failure encountered while determining the public IPv4
// address. The wrapped error is available via Unwrap for errors.Is/errors.As.
type IPCheckError struct {
	Err error
}

// Error implements the error interface for IPCheckError.
func (e *IPCheckError) Error() string {
	return fmt.Sprintf("ip check: %v", e.Err)
}

// Unwrap returns the underlying cause of the IP check failure.
func (e *IPCheckError) Unwrap() error { return e.Err }

// RecordUpdateError wraps a failure encountered while fetching or changing a
// single DNS A record, retaining the affected record name and zone.
type RecordUpdateError struct {
	// Record is the fully qualified name of the record that failed to update.
	Record string
	// Zone is the zone containing the affected record.
	Zone string
	Err  error
}

// Error implements the error interface for RecordUpdateError.
func (e *RecordUpdateError) Error() string {
	return fmt.Sprintf("update record %q in zone %q: %v", e.Record, e.Zone, e.Err)
}

// Unwrap returns the underlying cause of the record update failure.
func (e *RecordUpdateError) Unwrap() error { return e.Err }

// NotificationError wraps a failure encountered while delivering a message to
// an external notification service such as Discord or ntfy.
type NotificationError struct {
	// Service identifies the notification channel that failed (e.g. "discord").
	Service string
	Err     error
}

// Error implements the error interface for NotificationError.
func (e *NotificationError) Error() string {
	return fmt.Sprintf("%s notification: %v", e.Service, e.Err)
}

// Unwrap returns the underlying cause of the notification failure.
func (e *NotificationError) Unwrap() error { return e.Err }
