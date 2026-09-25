package main

import (
	"fmt"
	"strings"
)

// RRSet identifies a single DNS A record by its fully qualified name and zone.
type RRSet struct {
	// Name is the fully qualified record name, including the trailing dot.
	Name string
	// Zone is the PowerDNS zone that contains the record.
	Zone string
}

type rrsetSlice []RRSet

// String implements pflag.Value and returns the display form of the rrsets.
func (r *rrsetSlice) String() string {
	parts := make([]string, len(*r))
	for i, rr := range *r {
		parts[i] = rr.Name + "," + rr.Zone
	}
	return strings.Join(parts, "; ")
}

// Type implements pflag.Value and returns the expected value format.
func (r *rrsetSlice) Type() string {
	return "record,zone"
}

// Set implements pflag.Value, parsing a "record,zone" tuple and appending it.
func (r *rrsetSlice) Set(value string) error {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid rrset %q: expected format record,zone", value)
	}
	name := strings.TrimSpace(parts[0])
	zone := strings.TrimSpace(parts[1])
	if name == "" {
		return fmt.Errorf("invalid rrset %q: record name must not be empty", value)
	}
	if zone == "" {
		return fmt.Errorf("invalid rrset %q: zone must not be empty", value)
	}
	*r = append(*r, RRSet{Name: name, Zone: zone})
	return nil
}
