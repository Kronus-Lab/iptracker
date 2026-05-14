package main

import (
	"context"
	"net/http"

	"github.com/joeig/go-powerdns/v3"
)

type RecordsClient interface {
	Get(ctx context.Context, domain, name string, recordType *powerdns.RRType) ([]powerdns.RRset, error)
	Change(ctx context.Context, domain string, name string, recordType powerdns.RRType, ttl uint32, content []string, options ...func(*powerdns.RRset)) error
}

type PowerDNSRecordsClient struct {
	client *powerdns.Client
}

func (p *PowerDNSRecordsClient) Get(ctx context.Context, domain, name string, recordType *powerdns.RRType) ([]powerdns.RRset, error) {
	return p.client.Records.Get(ctx, domain, name, recordType)
}

func (p *PowerDNSRecordsClient) Change(ctx context.Context, domain string, name string, recordType powerdns.RRType, ttl uint32, content []string, options ...func(*powerdns.RRset)) error {
	return p.client.Records.Change(ctx, domain, name, recordType, ttl, content, options...)
}

func newRecordsClient(pdnsApiKey, pdnsApiUrl string, httpClient *http.Client) RecordsClient {
	client := powerdns.New(pdnsApiUrl, "localhost", powerdns.WithAPIKey(pdnsApiKey), powerdns.WithHTTPClient(httpClient))
	return &PowerDNSRecordsClient{client: client}
}
