package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotifyDiscord_Success(t *testing.T) {
	var receivedBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content-type, got %s", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := notifyDiscord("test message", server.URL+"/webhook", server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody["content"] != "test message" {
		t.Errorf("expected content 'test message', got %q", receivedBody["content"])
	}
}

func TestNotifyDiscord_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := server.Client()
	url := server.URL
	server.Close()

	err := notifyDiscord("test", url+"/webhook", client)
	if err == nil {
		t.Fatal("expected error when server is unreachable")
	}
}

func TestNotifyDiscord_BadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	err := notifyDiscord("test", server.URL+"/webhook", server.Client())
	if err == nil {
		t.Fatal("expected error for bad status code")
	}
	var nerr *NotificationError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected NotificationError, got %T", err)
	}
	if nerr.Service != "discord" {
		t.Errorf("expected service discord, got %s", nerr.Service)
	}
}

func TestNotifyNtfy_Success(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("expected text/plain content-type, got %s", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := notifyNtfy("test message", server.URL+"/mytopic", server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody != "test message" {
		t.Errorf("expected body 'test message', got %q", receivedBody)
	}
}

func TestNotifyNtfy_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	client := server.Client()
	url := server.URL
	server.Close()

	err := notifyNtfy("test", url+"/webhook", client)
	if err == nil {
		t.Fatal("expected error when server is unreachable")
	}
}

func TestNotifyNtfy_BadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := notifyNtfy("test", server.URL+"/webhook", server.Client())
	if err == nil {
		t.Fatal("expected error for bad status code")
	}
	var nerr *NotificationError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected NotificationError, got %T", err)
	}
	if nerr.Service != "ntfy" {
		t.Errorf("expected service ntfy, got %s", nerr.Service)
	}
}

func TestSendNotifications_NoWebhooks(t *testing.T) {
	err := sendNotifications("test", "", "", http.DefaultClient)
	if err != nil {
		t.Fatalf("expected nil when no webhooks configured, got %v", err)
	}
}

func TestSendNotifications_DiscordOnly(t *testing.T) {
	discordCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		discordCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := sendNotifications("test", server.URL+"/webhook", "", server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !discordCalled {
		t.Fatal("expected discord to be called")
	}
}

func TestSendNotifications_NtfyOnly(t *testing.T) {
	ntfyCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ntfyCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := sendNotifications("test", "", server.URL+"/mytopic", server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ntfyCalled {
		t.Fatal("expected ntfy to be called")
	}
}

func TestSendNotifications_Both(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := sendNotifications("test", server.URL+"/discord", server.URL+"/ntfy", server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
}

func TestSendNotifications_DiscordFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := sendNotifications("test", server.URL+"/webhook", "", server.Client())
	if err == nil {
		t.Fatal("expected error when discord fails")
	}
	var nerr *NotificationError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected NotificationError, got %T", err)
	}
}

func TestSendNotifications_NtfyFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := sendNotifications("test", "", server.URL+"/topic", server.Client())
	if err == nil {
		t.Fatal("expected error when ntfy fails")
	}
	var nerr *NotificationError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected NotificationError, got %T", err)
	}
}

func TestSendNotifications_BothFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := sendNotifications("test", server.URL+"/discord", server.URL+"/ntfy", server.Client())
	if err == nil {
		t.Fatal("expected error when both fail")
	}

	var discordErr, ntfyErr *NotificationError
	if !errors.As(err, &discordErr) {
		t.Fatalf("expected discord NotificationError in joined error, got %T", err)
	}
	if !errors.As(err, &ntfyErr) {
		t.Fatalf("expected ntfy NotificationError in joined error, got %T", err)
	}
}

func TestNotifyDiscord_PayloadFormat(t *testing.T) {
	var contentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := notifyDiscord("hello world", server.URL+"/webhook", server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected application/json content-type, got %s", contentType)
	}
}

func TestNotifyNtfy_PayloadFormat(t *testing.T) {
	var contentType string
	var bodyStr string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		bodyStr = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := notifyNtfy("hello world", server.URL+"/topic", server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("expected text/plain content-type, got %s", contentType)
	}
	if bodyStr != "hello world" {
		t.Errorf("expected body 'hello world', got %q", bodyStr)
	}
}
