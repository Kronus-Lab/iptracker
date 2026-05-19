package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/joeig/go-powerdns/v3"
)

func TestValidateWebhookURL_Empty(t *testing.T) {
	if err := validateWebhookURL(""); err != nil {
		t.Errorf("expected nil for empty URL, got %v", err)
	}
}

func TestValidateWebhookURL_HTTP(t *testing.T) {
	if err := validateWebhookURL("http://example.com/webhook"); err != nil {
		t.Errorf("expected nil for http URL, got %v", err)
	}
}

func TestValidateWebhookURL_HTTPS(t *testing.T) {
	if err := validateWebhookURL("https://example.com/webhook"); err != nil {
		t.Errorf("expected nil for https URL, got %v", err)
	}
}

func TestValidateWebhookURL_BadScheme(t *testing.T) {
	err := validateWebhookURL("ftp://example.com/webhook")
	if err == nil {
		t.Fatal("expected error for ftp scheme")
	}
	if !strings.Contains(err.Error(), "http or https") {
		t.Errorf("expected scheme error, got %v", err)
	}
}

func TestValidateWebhookURL_NoHost(t *testing.T) {
	err := validateWebhookURL("https://")
	if err == nil {
		t.Fatal("expected error for missing host")
	}
	if !strings.Contains(err.Error(), "host") {
		t.Errorf("expected host error, got %v", err)
	}
}

func TestValidateWebhookURL_InvalidURL(t *testing.T) {
	err := validateWebhookURL("://bad")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestRunCycle_IPChanged_SendsNotification(t *testing.T) {
	var ntfyBody string
	ntfyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		ntfyBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ntfyServer.Close()

	mock := &MockRecordsClient{
		GetResult: []powerdns.RRset{
			{Records: []powerdns.Record{{Content: strPtr("1.2.3.4")}}},
		},
	}

	updated, err := runCycle(
		context.Background(),
		"5.6.7.8",
		rrsetSlice{{Name: "myhost.", Zone: "example.com."}},
		mock,
		"",
		ntfyServer.URL+"/mytopic",
		ntfyServer.Client(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Fatal("expected updated to be true")
	}
	if !mock.ChangeCalled {
		t.Fatal("expected Change to be called since IP differs")
	}
	if !strings.Contains(ntfyBody, "5.6.7.8") {
		t.Errorf("expected ntfy notification to contain IP, got %q", ntfyBody)
	}
}

func TestRunCycle_IPUnchanged_NoNotification(t *testing.T) {
	notificationSent := false
	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notificationSent = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer discordServer.Close()

	mock := &MockRecordsClient{
		GetResult: []powerdns.RRset{
			{Records: []powerdns.Record{{Content: strPtr("1.2.3.4")}}},
		},
	}

	updated, err := runCycle(
		context.Background(),
		"1.2.3.4",
		rrsetSlice{{Name: "myhost.", Zone: "example.com."}},
		mock,
		discordServer.URL+"/webhook",
		"",
		discordServer.Client(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated {
		t.Fatal("expected updated to be false since IP unchanged")
	}
	if notificationSent {
		t.Fatal("expected no notification when IP unchanged")
	}
}

func TestRunCycle_UpdateError_SendsCriticalNotification(t *testing.T) {
	var ntfyBody string
	ntfyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		ntfyBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ntfyServer.Close()

	mock := &MockRecordsClient{
		GetError: fmt.Errorf("API error"),
	}

	_, err := runCycle(
		context.Background(),
		"5.6.7.8",
		rrsetSlice{{Name: "myhost.", Zone: "example.com."}},
		mock,
		"",
		ntfyServer.URL+"/mytopic",
		ntfyServer.Client(),
	)
	if err == nil {
		t.Fatal("expected error when update fails")
	}
	if !strings.Contains(ntfyBody, "CRITICAL") {
		t.Errorf("expected CRITICAL notification, got %q", ntfyBody)
	}
}

func TestRunCycle_UpdateError_NoWebhooks(t *testing.T) {
	mock := &MockRecordsClient{
		GetError: fmt.Errorf("API error"),
	}

	_, err := runCycle(
		context.Background(),
		"5.6.7.8",
		rrsetSlice{{Name: "myhost.", Zone: "example.com."}},
		mock,
		"",
		"",
		http.DefaultClient,
	)
	if err == nil {
		t.Fatal("expected error when update fails")
	}
}

func TestRunCycle_UpdateError_NotificationAlsoFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	mock := &MockRecordsClient{
		GetError: fmt.Errorf("API error"),
	}

	_, err := runCycle(
		context.Background(),
		"5.6.7.8",
		rrsetSlice{{Name: "myhost.", Zone: "example.com."}},
		mock,
		"",
		server.URL+"/mytopic",
		server.Client(),
	)
	if err == nil {
		t.Fatal("expected error when update fails")
	}
}

func TestRunCycle_NoUpdatesNoErrors_NoNotification(t *testing.T) {
	notificationSent := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notificationSent = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mock := &MockRecordsClient{
		GetResult: []powerdns.RRset{
			{Records: []powerdns.Record{{Content: strPtr("1.2.3.4")}}},
		},
	}

	updated, err := runCycle(
		context.Background(),
		"1.2.3.4",
		rrsetSlice{{Name: "myhost.", Zone: "example.com."}},
		mock,
		server.URL+"/discord",
		server.URL+"/ntfy",
		server.Client(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated {
		t.Fatal("expected updated to be false since IP unchanged")
	}
	if notificationSent {
		t.Fatal("expected no notification when IP unchanged and no errors")
	}
}

func TestRunCycle_NotificationError_DoesNotBlock(t *testing.T) {
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer badServer.Close()

	badClient := badServer.Client()
	closeServer := func() { badServer.Close() }
	closeServer()

	mock := &MockRecordsClient{
		GetResult: []powerdns.RRset{
			{Records: []powerdns.Record{{Content: strPtr("1.2.3.4")}}},
		},
	}

	updated, err := runCycle(
		context.Background(),
		"5.6.7.8",
		rrsetSlice{{Name: "myhost.", Zone: "example.com."}},
		mock,
		"http://127.0.0.1:1/webhook",
		"",
		badClient,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Fatal("expected updated to be true even when notification fails")
	}
}
