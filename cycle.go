package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
)

func runCycle(ctx context.Context, liveIP string, recordSets rrsetSlice, recordsClient RecordsClient, discord string, ntfyWebhook string, httpClient *http.Client) (bool, error) {
	var wg sync.WaitGroup
	var hasError atomic.Bool
	var updated atomic.Bool

	for i := range recordSets {
		wg.Add(1)
		rrset := recordSets[i]
		go func() {
			defer wg.Done()
			changed, err := updateRecord(ctx, rrset, liveIP, recordsClient)
			if err != nil {
				slog.Error("failed to update record", "error", err)
				hasError.Store(true)
				return
			}
			if changed {
				updated.Store(true)
			}
		}()
	}
	wg.Wait()

	if hasError.Load() {
		slog.Error("DNS update cycle had errors", "ip", liveIP)
		if notifyErr := sendNotifications("CRITICAL: DNS update failed — live IP is "+liveIP, discord, ntfyWebhook, httpClient); notifyErr != nil {
			slog.Warn("failed to send error notification", "error", notifyErr)
		}
		return updated.Load(), fmt.Errorf("DNS update cycle had errors for IP %s", liveIP)
	}

	if updated.Load() {
		if notifyErr := sendNotifications("IP address updated: "+liveIP, discord, ntfyWebhook, httpClient); notifyErr != nil {
			slog.Warn("failed to send update notification", "error", notifyErr)
		}
	}
	return updated.Load(), nil
}

func validateWebhookURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("webhook URL must use http or https scheme, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("webhook URL must have a host")
	}
	return nil
}
