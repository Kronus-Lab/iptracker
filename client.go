package main

import (
	"context"
	"net/http"

	"github.com/joeig/go-powerdns/v3"
)

// RecordsClient abstracts the PowerDNS operations required to reconcile a
// single A record. It is implemented by PowerDNSRecordsClient for live use and
// by the mock in unit tests.
type RecordsClient interface {
	// Get fetches the rrsets matching the given record name within the zone.
	Get(ctx context.Context, domain, name string, recordType *powerdns.RRType) ([]powerdns.RRset, error)
	// Change replaces the content of the record, which is 1 or more content values within the zone.
	Change(ctx context.Context, domain string, name string, recordType powerdns.RRType, ttl uint32, content []string, options ...func(*powerdns.RRset)) error
}

// PowerDNSRecordsClient implements RecordsClient against a live PowerDNS
// installation through the go-powerdns library.
type PowerDNSRecordsClient struct {
	client *powerdns.Client
}

// Get implements RecordsClient.Get.
func (p *PowerDNSRecordsClient) Get(ctx context.Context, domain, name string, recordType *powerdns.RRType) ([]powerdns.RRset, error) {
	return p.client.Records.Get(ctx, domain, name, recordType)
}

// Change implements RecordsClient.Change.
func (p *PowerDNSRecordsClient) Change(ctx context.Context, domain string, name string, recordType powerdns.RRType, ttl uint32, content []string, options ...func(*powerdns.RRset)) error {
	return p.client.Records.Change(ctx, domain, name, recordType, ttl, content, options...)
}

// newRecordsClient builds a RecordsClient bound to the PowerDNS API endpoint
// and key at the given URL, using the provided HTTP client for all requests.
func newRecordsClient(pdnsApiKey, pdnsApiUrl string, httpClient *http.Client) RecordsClient {
	client := powerdns.New(pdnsApiUrl, "localhost", powerdns.WithAPIKey(pdnsApiKey), powerdns.WithHTTPClient(httpClient))
	return &PowerDNSRecordsClient{client: client}
}
