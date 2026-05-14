package main

import (
	"fmt"
	"strings"
)

type RRSet struct {
	Name string
	Zone string
}

type rrsetSlice []RRSet

func (r *rrsetSlice) String() string {
	parts := make([]string, len(*r))
	for i, rr := range *r {
		parts[i] = rr.Name + "," + rr.Zone
	}
	return strings.Join(parts, "; ")
}

func (r *rrsetSlice) Type() string {
	return "record,zone"
}

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
