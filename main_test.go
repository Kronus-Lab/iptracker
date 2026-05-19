package main

import (
	"context"

	"github.com/joeig/go-powerdns/v3"
)

type MockRecordsClient struct {
	GetResult    []powerdns.RRset
	GetError     error
	ChangeError  error
	GetCalled    bool
	ChangeCalled bool
	GetCalls     []getCall
	ChangeCalls  []changeCall
}

type getCall struct {
	domain     string
	name       string
	recordType *powerdns.RRType
}

type changeCall struct {
	domain     string
	name       string
	recordType powerdns.RRType
	ttl        uint32
	content    []string
}

func (m *MockRecordsClient) Get(ctx context.Context, domain, name string, recordType *powerdns.RRType) ([]powerdns.RRset, error) {
	m.GetCalled = true
	m.GetCalls = append(m.GetCalls, getCall{domain: domain, name: name, recordType: recordType})
	return m.GetResult, m.GetError
}

func (m *MockRecordsClient) Change(ctx context.Context, domain string, name string, recordType powerdns.RRType, ttl uint32, content []string, options ...func(*powerdns.RRset)) error {
	m.ChangeCalled = true
	m.ChangeCalls = append(m.ChangeCalls, changeCall{domain: domain, name: name, recordType: recordType, ttl: ttl, content: content})
	return m.ChangeError
}

func strPtr(s string) *string {
	return &s
}
