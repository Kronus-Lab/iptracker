package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

const maxIPResponseSize = 45

const defaultIPCheckURL = "https://ifconfig.me"

// getLiveIP fetches the public IPv4 address from the provided URL, validates that the response is a syntactically valid IPv4 address, and returns it.
// On failure it returns an empty string and an *IPCheckError describing the cause.
func getLiveIP(ctx context.Context, httpClient *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", &IPCheckError{Err: fmt.Errorf("create request: %w", err)}
	}
	req.Header.Set("User-Agent", "curl/8.12.1")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", &IPCheckError{Err: fmt.Errorf("reach ifconfig.me: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", &IPCheckError{Err: fmt.Errorf("non-200 status: %s", resp.Status)}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxIPResponseSize))
	if err != nil {
		return "", &IPCheckError{Err: fmt.Errorf("read response: %w", err)}
	}

	liveIP := strings.TrimSpace(string(body))
	if ip := net.ParseIP(liveIP); ip == nil || ip.To4() == nil {
		return "", &IPCheckError{Err: fmt.Errorf("not a valid IPv4 address: %q", liveIP)}
	}

	return liveIP, nil
}
