package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/joeig/go-powerdns/v3"
)

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
