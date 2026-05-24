package main

import (
	"errors"
	"fmt"
	"testing"
)

func TestIPCheckError_Error(t *testing.T) {
	err := &IPCheckError{Err: fmt.Errorf("some failure")}
	msg := err.Error()
	expected := "ip check: some failure"
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

func TestIPCheckError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	err := &IPCheckError{Err: inner}
	if !errors.Is(err, inner) {
		t.Fatal("expected errors.Is to match inner error")
	}
}

func TestRecordUpdateError_Error(t *testing.T) {
	err := &RecordUpdateError{Record: "myhost", Zone: "example.com", Err: fmt.Errorf("fail")}
	msg := err.Error()
	expected := `update record "myhost" in zone "example.com": fail`
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

func TestRecordUpdateError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	err := &RecordUpdateError{Record: "myhost", Zone: "example.com", Err: inner}
	if !errors.Is(err, inner) {
		t.Fatal("expected errors.Is to match inner error")
	}
}

func TestNotificationError_Error(t *testing.T) {
	err := &NotificationError{Service: "discord", Err: fmt.Errorf("timeout")}
	msg := err.Error()
	expected := "discord notification: timeout"
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

func TestNotificationError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	err := &NotificationError{Service: "ntfy", Err: inner}
	if !errors.Is(err, inner) {
		t.Fatal("expected errors.Is to match inner error")
	}
}
