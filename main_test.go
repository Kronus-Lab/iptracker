package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

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

func TestUpdateRecord_IPChanged(t *testing.T) {
	mock := &MockRecordsClient{
		GetResult: []powerdns.RRset{
			{
				Records: []powerdns.Record{
					{Content: strPtr("1.2.3.4")},
				},
			},
		},
	}

	changed, err := updateRecord(context.Background(), RRSet{Name: "myhost", Zone: "example.com"}, "5.6.7.8", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed to be true")
	}
	if !mock.GetCalled {
		t.Fatal("expected Get to be called")
	}
	if !mock.ChangeCalled {
		t.Fatal("expected Change to be called")
	}
	if len(mock.ChangeCalls) != 1 {
		t.Fatalf("expected 1 Change call, got %d", len(mock.ChangeCalls))
	}
	call := mock.ChangeCalls[0]
	if call.domain != "example.com" {
		t.Errorf("expected domain example.com, got %s", call.domain)
	}
	if call.name != "myhost" {
		t.Errorf("expected name myhost, got %s", call.name)
	}
	if call.ttl != 60 {
		t.Errorf("expected ttl 60, got %d", call.ttl)
	}
	if len(call.content) != 1 || call.content[0] != "5.6.7.8" {
		t.Errorf("expected content [5.6.7.8], got %v", call.content)
	}
}

func TestUpdateRecord_IPUnchanged(t *testing.T) {
	mock := &MockRecordsClient{
		GetResult: []powerdns.RRset{
			{
				Records: []powerdns.Record{
					{Content: strPtr("1.2.3.4")},
				},
			},
		},
	}

	changed, err := updateRecord(context.Background(), RRSet{Name: "myhost", Zone: "example.com"}, "1.2.3.4", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Fatal("expected changed to be false since IP unchanged")
	}
	if !mock.GetCalled {
		t.Fatal("expected Get to be called")
	}
	if mock.ChangeCalled {
		t.Fatal("expected Change NOT to be called when IP is unchanged")
	}
}

func TestUpdateRecord_GetError(t *testing.T) {
	mock := &MockRecordsClient{
		GetError: fmt.Errorf("API error"),
	}

	changed, err := updateRecord(context.Background(), RRSet{Name: "myhost", Zone: "example.com"}, "5.6.7.8", mock)
	if err == nil {
		t.Fatal("expected error on Get failure")
	}
	if changed {
		t.Fatal("expected changed to be false on Get failure")
	}
	if mock.ChangeCalled {
		t.Fatal("expected Change NOT to be called on Get failure")
	}
	var rerr *RecordUpdateError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected RecordUpdateError, got %T", err)
	}
	if rerr.Record != "myhost" || rerr.Zone != "example.com" {
		t.Errorf("expected record=myhost zone=example.com, got record=%s zone=%s", rerr.Record, rerr.Zone)
	}
}

func TestUpdateRecord_MultipleRRSets(t *testing.T) {
	mock := &MockRecordsClient{
		GetResult: []powerdns.RRset{
			{Records: []powerdns.Record{{Content: strPtr("1.2.3.4")}}},
			{Records: []powerdns.Record{{Content: strPtr("5.6.7.8")}}},
		},
	}

	changed, err := updateRecord(context.Background(), RRSet{Name: "myhost", Zone: "example.com"}, "9.9.9.9", mock)
	if err == nil {
		t.Fatal("expected error when multiple rrsets returned")
	}
	if changed {
		t.Fatal("expected changed to be false")
	}
}

func TestUpdateRecord_MultipleRecordsInRRSet(t *testing.T) {
	mock := &MockRecordsClient{
		GetResult: []powerdns.RRset{
			{
				Records: []powerdns.Record{
					{Content: strPtr("1.2.3.4")},
					{Content: strPtr("5.6.7.8")},
				},
			},
		},
	}

	changed, err := updateRecord(context.Background(), RRSet{Name: "myhost", Zone: "example.com"}, "9.9.9.9", mock)
	if err == nil {
		t.Fatal("expected error when multiple records in rrset")
	}
	if changed {
		t.Fatal("expected changed to be false")
	}
}

func TestUpdateRecord_ChangeError(t *testing.T) {
	mock := &MockRecordsClient{
		GetResult: []powerdns.RRset{
			{
				Records: []powerdns.Record{
					{Content: strPtr("1.2.3.4")},
				},
			},
		},
		ChangeError: fmt.Errorf("change failed"),
	}

	changed, err := updateRecord(context.Background(), RRSet{Name: "myhost", Zone: "example.com"}, "5.6.7.8", mock)
	if err == nil {
		t.Fatal("expected error when Change fails")
	}
	if changed {
		t.Fatal("expected changed to be false on Change failure")
	}
}

func TestUpdateRecord_EmptyRRSets(t *testing.T) {
	mock := &MockRecordsClient{
		GetResult: []powerdns.RRset{},
	}

	changed, err := updateRecord(context.Background(), RRSet{Name: "myhost", Zone: "example.com"}, "5.6.7.8", mock)
	if err == nil {
		t.Fatal("expected error when no rrsets returned")
	}
	if changed {
		t.Fatal("expected changed to be false")
	}
}

func TestUpdateRecord_NilContent(t *testing.T) {
	mock := &MockRecordsClient{
		GetResult: []powerdns.RRset{
			{
				Records: []powerdns.Record{
					{Content: nil},
				},
			},
		},
	}

	changed, err := updateRecord(context.Background(), RRSet{Name: "myhost", Zone: "example.com"}, "5.6.7.8", mock)
	if err == nil {
		t.Fatal("expected error when Content is nil")
	}
	if changed {
		t.Fatal("expected changed to be false")
	}
	if mock.ChangeCalled {
		t.Fatal("expected Change NOT to be called when Content is nil")
	}
}

func TestIPCheckError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	err := &IPCheckError{Err: inner}
	if !errors.Is(err, inner) {
		t.Fatal("expected errors.Is to match inner error")
	}
}

func TestRecordUpdateError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	err := &RecordUpdateError{Record: "myhost", Zone: "example.com", Err: inner}
	if !errors.Is(err, inner) {
		t.Fatal("expected errors.Is to match inner error")
	}
}

func TestNotificationError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	err := &NotificationError{Service: "discord", Err: inner}
	if !errors.Is(err, inner) {
		t.Fatal("expected errors.Is to match inner error")
	}
}

func TestRRSetSlice_ValidInput(t *testing.T) {
	var r rrsetSlice
	if err := r.Set("myhost,example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := r.Set(" otherhost , otherzone.io "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r) != 2 {
		t.Fatalf("expected 2 rrsets, got %d", len(r))
	}
	if r[0].Name != "myhost" || r[0].Zone != "example.com" {
		t.Errorf("expected myhost/example.com, got %s/%s", r[0].Name, r[0].Zone)
	}
	if r[1].Name != "otherhost" || r[1].Zone != "otherzone.io" {
		t.Errorf("expected otherhost/otherzone.io, got %s/%s", r[1].Name, r[1].Zone)
	}
}

func TestRRSetSlice_NoComma(t *testing.T) {
	var r rrsetSlice
	err := r.Set("myhost")
	if err == nil {
		t.Fatal("expected error for input without comma")
	}
}

func TestRRSetSlice_EmptyName(t *testing.T) {
	var r rrsetSlice
	err := r.Set(",example.com")
	if err == nil {
		t.Fatal("expected error for empty record name")
	}
}

func TestRRSetSlice_EmptyZone(t *testing.T) {
	var r rrsetSlice
	err := r.Set("myhost,")
	if err == nil {
		t.Fatal("expected error for empty zone")
	}
}

func TestRRSetSlice_CommaInZone(t *testing.T) {
	var r rrsetSlice
	if err := r.Set("myhost,example.com,extra"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r[0].Zone != "example.com,extra" {
		t.Errorf("expected zone 'example.com,extra', got %s", r[0].Zone)
	}
}

func TestRRSetSlice_Type(t *testing.T) {
	var r rrsetSlice
	if r.Type() != "record,zone" {
		t.Errorf("expected type 'record,zone', got %s", r.Type())
	}
}

func TestRRSetSlice_String(t *testing.T) {
	r := rrsetSlice{{Name: "a", Zone: "b.com"}, {Name: "c", Zone: "d.org"}}
	expected := "a,b.com; c,d.org"
	if r.String() != expected {
		t.Errorf("expected %q, got %q", expected, r.String())
	}
}
