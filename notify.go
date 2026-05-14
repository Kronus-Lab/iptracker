package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

func notifyDiscord(message string, webhook string, httpClient *http.Client) error {
	payload, err := json.Marshal(map[string]string{"content": message})
	if err != nil {
		return &NotificationError{Service: "discord", Err: fmt.Errorf("marshal payload: %w", err)}
	}
	resp, err := httpClient.Post(webhook, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return &NotificationError{Service: "discord", Err: err}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return &NotificationError{Service: "discord", Err: fmt.Errorf("unexpected status: %s", resp.Status)}
	}
	return nil
}

func notifyNtfy(message string, webhook string, httpClient *http.Client) error {
	resp, err := httpClient.Post(webhook, "text/plain", strings.NewReader(message))
	if err != nil {
		return &NotificationError{Service: "ntfy", Err: err}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return &NotificationError{Service: "ntfy", Err: fmt.Errorf("unexpected status: %s", resp.Status)}
	}
	return nil
}

func sendNotifications(message string, discord string, ntfy string, httpClient *http.Client) error {
	var errs []error
	if discord != "" {
		if err := notifyDiscord(message, discord, httpClient); err != nil {
			slog.Warn("discord notification failed", "error", err)
			errs = append(errs, err)
		}
	}
	if ntfy != "" {
		if err := notifyNtfy(message, ntfy, httpClient); err != nil {
			slog.Warn("ntfy notification failed", "error", err)
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
