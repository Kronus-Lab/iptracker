package main

import (
	"testing"
)

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
