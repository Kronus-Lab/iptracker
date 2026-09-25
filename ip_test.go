package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

type mockTransport struct {
	statusCode int
	body       string
	bodyReader io.ReadCloser
	err        error
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.err != nil {
		return nil, t.err
	}
	body := t.bodyReader
	if body == nil {
		body = io.NopCloser(strings.NewReader(t.body))
	}
	return &http.Response{
		StatusCode: t.statusCode,
		Status:     http.StatusText(t.statusCode),
		Body:       body,
		Header:     make(http.Header),
	}, nil
}

func TestGetLiveIP_Success(t *testing.T) {
	client := &http.Client{Transport: &mockTransport{statusCode: 200, body: "1.2.3.4"}}
	ip, err := getLiveIP(context.Background(), client, defaultIPCheckURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4, got %s", ip)
	}
}

func TestGetLiveIP_WhitespaceTrimmed(t *testing.T) {
	client := &http.Client{Transport: &mockTransport{statusCode: 200, body: "  10.0.0.1\n  "}}
	ip, err := getLiveIP(context.Background(), client, defaultIPCheckURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %s", ip)
	}
}

func TestGetLiveIP_Non200Status(t *testing.T) {
	client := &http.Client{Transport: &mockTransport{statusCode: 500, body: "error"}}
	_, err := getLiveIP(context.Background(), client, defaultIPCheckURL)
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
	var ipErr *IPCheckError
	if !errors.As(err, &ipErr) {
		t.Fatalf("expected IPCheckError, got %T", err)
	}
}

func TestGetLiveIP_InvalidIPv4(t *testing.T) {
	client := &http.Client{Transport: &mockTransport{statusCode: 200, body: "not-an-ip"}}
	_, err := getLiveIP(context.Background(), client, defaultIPCheckURL)
	if err == nil {
		t.Fatal("expected error for invalid IP")
	}
	var ipErr *IPCheckError
	if !errors.As(err, &ipErr) {
		t.Fatalf("expected IPCheckError, got %T", err)
	}
}

func TestGetLiveIP_IPv6Rejected(t *testing.T) {
	client := &http.Client{Transport: &mockTransport{statusCode: 200, body: "::1"}}
	_, err := getLiveIP(context.Background(), client, defaultIPCheckURL)
	if err == nil {
		t.Fatal("expected error for IPv6 address")
	}
}

func TestGetLiveIP_RequestError(t *testing.T) {
	client := &http.Client{Transport: &mockTransport{err: &net.DNSError{Err: "dns failure"}}}
	_, err := getLiveIP(context.Background(), client, defaultIPCheckURL)
	if err == nil {
		t.Fatal("expected error for request failure")
	}
	var ipErr *IPCheckError
	if !errors.As(err, &ipErr) {
		t.Fatalf("expected IPCheckError, got %T", err)
	}
}

func TestGetLiveIP_InvalidURL(t *testing.T) {
	client := &http.Client{Transport: &mockTransport{statusCode: 200, body: "1.2.3.4"}}
	_, err := getLiveIP(context.Background(), client, "://bad")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	var ipErr *IPCheckError
	if !errors.As(err, &ipErr) {
		t.Fatalf("expected IPCheckError, got %T", err)
	}
	if ipErr.Err == nil {
		t.Fatal("expected IPCheckError.Err to be non-nil")
	}
}

func TestGetLiveIP_UserAgentSet(t *testing.T) {
	var capturedUA string
	transport := &roundTripCapturer{
		statusCode: 200,
		body:       "1.2.3.4",
		capture: func(req *http.Request) {
			capturedUA = req.Header.Get("User-Agent")
		},
	}
	client := &http.Client{Transport: transport}
	_, err := getLiveIP(context.Background(), client, defaultIPCheckURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedUA != "curl/8.12.1" {
		t.Errorf("expected User-Agent 'curl/8.12.1', got %q", capturedUA)
	}
}

func TestGetLiveIP_MethodIsGET(t *testing.T) {
	var capturedMethod string
	transport := &roundTripCapturer{
		statusCode: 200,
		body:       "1.2.3.4",
		capture: func(req *http.Request) {
			capturedMethod = req.Method
		},
	}
	client := &http.Client{Transport: transport}
	_, err := getLiveIP(context.Background(), client, defaultIPCheckURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedMethod != http.MethodGet {
		t.Errorf("expected GET method, got %q", capturedMethod)
	}
}

type readErrReader struct{}

func (r *readErrReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("read error")
}

func (r *readErrReader) Close() error {
	return nil
}

func TestGetLiveIP_ReadError(t *testing.T) {
	client := &http.Client{Transport: &mockTransport{
		statusCode: 200,
		body:       "",
		bodyReader: io.NopCloser(&readErrReader{}),
	}}
	_, err := getLiveIP(context.Background(), client, defaultIPCheckURL)
	if err == nil {
		t.Fatal("expected error for read failure")
	}
	var ipErr *IPCheckError
	if !errors.As(err, &ipErr) {
		t.Fatalf("expected IPCheckError, got %T", err)
	}
}

type roundTripCapturer struct {
	statusCode int
	body       string
	capture    func(*http.Request)
}

func (t *roundTripCapturer) RoundTrip(req *http.Request) (*http.Response, error) {
	t.capture(req)
	return &http.Response{
		StatusCode: t.statusCode,
		Status:     http.StatusText(t.statusCode),
		Body:       io.NopCloser(strings.NewReader(t.body)),
		Header:     make(http.Header),
	}, nil
}
