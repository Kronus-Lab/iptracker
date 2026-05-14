package main

import "fmt"

type IPCheckError struct {
	Err error
}

func (e *IPCheckError) Error() string {
	return fmt.Sprintf("ip check: %v", e.Err)
}

func (e *IPCheckError) Unwrap() error { return e.Err }

type RecordUpdateError struct {
	Record string
	Zone   string
	Err    error
}

func (e *RecordUpdateError) Error() string {
	return fmt.Sprintf("update record %q in zone %q: %v", e.Record, e.Zone, e.Err)
}

func (e *RecordUpdateError) Unwrap() error { return e.Err }

type NotificationError struct {
	Service string
	Err     error
}

func (e *NotificationError) Error() string {
	return fmt.Sprintf("%s notification: %v", e.Service, e.Err)
}

func (e *NotificationError) Unwrap() error { return e.Err }
