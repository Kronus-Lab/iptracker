//go:build integration

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joeig/go-powerdns/v3"
)

const (
	defaultPDNSURL = "http://127.0.0.1:8081"
	defaultPDNSKey = "testapikey"
	defaultNtfyURL = "http://127.0.0.1:8082"
	testZone       = "test.example.net."
	testRecord     = "myhost.test.example.net."
	testInitialIP  = "1.2.3.4"
	testNewIP      = "5.6.7.8"
	ntfyTopic      = "iptracker-test"
	pollTimeout    = 15 * time.Second
	pollInterval   = 500 * time.Millisecond
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDiscordWebhook() string {
	return os.Getenv("DISCORD_WEBHOOK_URL")
}

func newTestHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

func waitForPowerDNS(t *testing.T, apiKey, url string) {
	t.Helper()
	client := newTestHTTPClient()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url+"/api/v1/servers/localhost", nil)
		req.Header.Set("X-API-Key", apiKey)
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return
		}
		if err == nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("timed out waiting for PowerDNS API")
}

func waitForNtfy(t *testing.T, url string) {
	t.Helper()
	client := newTestHTTPClient()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url + "/v1/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return
		}
		if err == nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("timed out waiting for ntfy")
}

func seedZone(t *testing.T, apiKey, pdnsURL, zoneName string) {
	t.Helper()
	client := newTestHTTPClient()
	payload := map[string]interface{}{
		"name":        zoneName,
		"kind":        "Native",
		"masters":     []string{},
		"nameservers": []string{"ns1." + zoneName, "ns2." + zoneName},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, pdnsURL+"/api/v1/servers/localhost/zones", bytes.NewReader(body))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to create zone: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("failed to create zone: status %s, body: %s", resp.Status, respBody)
	}
}

func seedRecord(t *testing.T, recordsClient RecordsClient, zone, name, ip string) {
	t.Helper()
	ctx := context.Background()
	err := recordsClient.Change(ctx, zone, name, powerdns.RRTypeA, 60, []string{ip})
	if err != nil {
		t.Fatalf("failed to seed record: %v", err)
	}
}

func deleteZone(t *testing.T, apiKey, pdnsURL, zoneName string) {
	t.Helper()
	client := newTestHTTPClient()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, pdnsURL+"/api/v1/servers/localhost/zones/"+zoneName, nil)
	req.Header.Set("X-API-Key", apiKey)
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("warning: failed to delete zone: %v", err)
		return
	}
	resp.Body.Close()
}

func fetchRecordIP(t *testing.T, recordsClient RecordsClient, zone, name string) string {
	t.Helper()
	ctx := context.Background()
	rrsets, err := recordsClient.Get(ctx, zone, name, powerdns.RRTypePtr(powerdns.RRTypeA))
	if err != nil {
		t.Fatalf("failed to fetch record: %v", err)
	}
	if len(rrsets) == 0 || len(rrsets[0].Records) == 0 || rrsets[0].Records[0].Content == nil {
		t.Fatal("no A record found")
	}
	return *rrsets[0].Records[0].Content
}

func waitForNtfyMessage(t *testing.T, ntfyURL, topic, substring string) {
	t.Helper()
	client := newTestHTTPClient()
	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, ntfyURL+"/"+topic+"/json?since=5m&poll=1", nil)
		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, substring) && strings.Contains(line, `"event":"message"`) {
				resp.Body.Close()
				return
			}
		}
		resp.Body.Close()
		time.Sleep(pollInterval)
	}
	t.Fatalf("timed out waiting for ntfy message containing %q", substring)
}

func setupIntegration(t *testing.T) (RecordsClient, string, string, func()) {
	t.Helper()

	pdnsURL := envOrDefault("PDNS_URL", defaultPDNSURL)
	pdnsKey := envOrDefault("PDNS_API_KEY", defaultPDNSKey)
	ntfyURL := envOrDefault("NTFY_URL", defaultNtfyURL)

	waitForPowerDNS(t, pdnsKey, pdnsURL)
	waitForNtfy(t, ntfyURL)

	seedZone(t, pdnsKey, pdnsURL, testZone)

	httpClient := newTestHTTPClient()
	recordsClient := newRecordsClient(pdnsKey, pdnsURL, httpClient)

	seedRecord(t, recordsClient, testZone, testRecord, testInitialIP)

	cleanup := func() {
		deleteZone(t, pdnsKey, pdnsURL, testZone)
	}

	return recordsClient, pdnsURL, ntfyURL, cleanup
}

func TestIntegration_PowerDNS_UpdateRecord(t *testing.T) {
	recordsClient, _, _, cleanup := setupIntegration(t)
	defer cleanup()

	changed, err := updateRecord(context.Background(), RRSet{Name: testRecord, Zone: testZone}, testNewIP, recordsClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed to be true")
	}

	got := fetchRecordIP(t, recordsClient, testZone, testRecord)
	if got != testNewIP {
		t.Fatalf("expected record IP %s, got %s", testNewIP, got)
	}
}

func TestIntegration_PowerDNS_IPUnchanged(t *testing.T) {
	recordsClient, _, _, cleanup := setupIntegration(t)
	defer cleanup()

	changed, err := updateRecord(context.Background(), RRSet{Name: testRecord, Zone: testZone}, testInitialIP, recordsClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Fatal("expected changed to be false since IP unchanged")
	}

	got := fetchRecordIP(t, recordsClient, testZone, testRecord)
	if got != testInitialIP {
		t.Fatalf("expected record IP unchanged at %s, got %s", testInitialIP, got)
	}
}

func TestIntegration_PowerDNS_ChangeError(t *testing.T) {
	recordsClient, _, _, cleanup := setupIntegration(t)
	defer cleanup()

	_, err := updateRecord(context.Background(), RRSet{Name: testRecord, Zone: "nonexistent.zone."}, testNewIP, recordsClient)
	if err == nil {
		t.Fatal("expected error when zone doesn't exist")
	}
}

func TestIntegration_Ntfy_SuccessNotification(t *testing.T) {
	recordsClient, _, ntfyURL, cleanup := setupIntegration(t)
	defer cleanup()

	ntfyWebhook := ntfyURL + "/" + ntfyTopic
	httpClient := newTestHTTPClient()

	updated, err := runCycle(context.Background(), testNewIP, rrsetSlice{{Name: testRecord, Zone: testZone}}, recordsClient, "", ntfyWebhook, httpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Fatal("expected updated to be true")
	}

	waitForNtfyMessage(t, ntfyURL, ntfyTopic, "IP address updated: "+testNewIP)
}

func TestIntegration_Ntfy_CriticalNotification(t *testing.T) {
	recordsClient, _, ntfyURL, cleanup := setupIntegration(t)
	defer cleanup()

	ntfyWebhook := ntfyURL + "/" + ntfyTopic
	httpClient := newTestHTTPClient()

	_, err := runCycle(context.Background(), testNewIP, rrsetSlice{{Name: testRecord, Zone: "nonexistent.zone."}}, recordsClient, "", ntfyWebhook, httpClient)
	if err == nil {
		t.Fatal("expected error for nonexistent zone")
	}

	waitForNtfyMessage(t, ntfyURL, ntfyTopic, "CRITICAL: DNS update failed")
}

func TestIntegration_Discord_SuccessNotification(t *testing.T) {
	webhook := getDiscordWebhook()
	if webhook == "" {
		t.Skip("DISCORD_WEBHOOK_URL not set, skipping Discord integration test")
	}

	recordsClient, _, _, cleanup := setupIntegration(t)
	defer cleanup()

	httpClient := newTestHTTPClient()

	updated, err := runCycle(context.Background(), testNewIP, rrsetSlice{{Name: testRecord, Zone: testZone}}, recordsClient, webhook, "", httpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Fatal("expected updated to be true")
	}
}

func TestIntegration_Discord_CriticalNotification(t *testing.T) {
	webhook := getDiscordWebhook()
	if webhook == "" {
		t.Skip("DISCORD_WEBHOOK_URL not set, skipping Discord integration test")
	}

	recordsClient, _, _, cleanup := setupIntegration(t)
	defer cleanup()

	httpClient := newTestHTTPClient()

	_, err := runCycle(context.Background(), testNewIP, rrsetSlice{{Name: testRecord, Zone: "nonexistent.zone."}}, recordsClient, webhook, "", httpClient)
	if err == nil {
		t.Fatal("expected error for nonexistent zone")
	}
}

func TestIntegration_FullCycle(t *testing.T) {
	recordsClient, _, ntfyURL, cleanup := setupIntegration(t)
	defer cleanup()

	got := fetchRecordIP(t, recordsClient, testZone, testRecord)
	if got != testInitialIP {
		t.Fatalf("expected initial IP %s, got %s", testInitialIP, got)
	}

	ntfyWebhook := ntfyURL + "/" + ntfyTopic
	discordWebhook := getDiscordWebhook()
	httpClient := newTestHTTPClient()

	updated, err := runCycle(context.Background(), testNewIP, rrsetSlice{{Name: testRecord, Zone: testZone}}, recordsClient, discordWebhook, ntfyWebhook, httpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Fatal("expected updated to be true after IP change")
	}

	got = fetchRecordIP(t, recordsClient, testZone, testRecord)
	if got != testNewIP {
		t.Fatalf("expected updated IP %s, got %s", testNewIP, got)
	}

	waitForNtfyMessage(t, ntfyURL, ntfyTopic, "IP address updated: "+testNewIP)

	updated, err = runCycle(context.Background(), testNewIP, rrsetSlice{{Name: testRecord, Zone: testZone}}, recordsClient, discordWebhook, ntfyWebhook, httpClient)
	if err != nil {
		t.Fatalf("unexpected error on second cycle: %v", err)
	}
	if updated {
		t.Fatal("expected updated to be false on second cycle since IP unchanged")
	}
}

func TestIntegration_MultipleRRSets(t *testing.T) {
	pdnsURL := envOrDefault("PDNS_URL", defaultPDNSURL)
	pdnsKey := envOrDefault("PDNS_API_KEY", defaultPDNSKey)

	waitForPowerDNS(t, pdnsKey, pdnsURL)

	zone1 := "multi1.example.net."
	zone2 := "multi2.example.net."
	seedZone(t, pdnsKey, pdnsURL, zone1)
	seedZone(t, pdnsKey, pdnsURL, zone2)
	defer deleteZone(t, pdnsKey, pdnsURL, zone1)
	defer deleteZone(t, pdnsKey, pdnsURL, zone2)

	httpClient := newTestHTTPClient()
	recordsClient := newRecordsClient(pdnsKey, pdnsURL, httpClient)
	seedRecord(t, recordsClient, zone1, "test."+zone1, testInitialIP)
	seedRecord(t, recordsClient, zone2, "test."+zone2, testInitialIP)

	ntfyURL := envOrDefault("NTFY_URL", defaultNtfyURL)
	ntfyWebhook := ntfyURL + "/" + ntfyTopic

	updated, err := runCycle(context.Background(), testNewIP, rrsetSlice{
		{Name: "test." + zone1, Zone: zone1},
		{Name: "test." + zone2, Zone: zone2},
	}, recordsClient, "", ntfyWebhook, httpClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Fatal("expected updated to be true after updating multiple zones")
	}

	got1 := fetchRecordIP(t, recordsClient, zone1, "test."+zone1)
	got2 := fetchRecordIP(t, recordsClient, zone2, "test."+zone2)
	if got1 != testNewIP {
		t.Fatalf("expected zone1 IP %s, got %s", testNewIP, got1)
	}
	if got2 != testNewIP {
		t.Fatalf("expected zone2 IP %s, got %s", testNewIP, got2)
	}

	waitForNtfyMessage(t, ntfyURL, ntfyTopic, "IP address updated: "+testNewIP)
}
