package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/joeig/go-powerdns/v3"
)

func updateRecord(ctx context.Context, rrset RRSet, liveIP string, recordsClient RecordsClient) (bool, error) {
	currentIP, err := recordsClient.Get(ctx, rrset.Zone, rrset.Name, powerdns.RRTypePtr(powerdns.RRTypeA))
	if err != nil {
		return false, &RecordUpdateError{Record: rrset.Name, Zone: rrset.Zone, Err: fmt.Errorf("fetch dns record: %w", err)}
	}
	if len(currentIP) != 1 {
		return false, &RecordUpdateError{Record: rrset.Name, Zone: rrset.Zone, Err: fmt.Errorf("unexpected API response: %d rrsets", len(currentIP))}
	}
	if len(currentIP[0].Records) != 1 {
		return false, &RecordUpdateError{Record: rrset.Name, Zone: rrset.Zone, Err: fmt.Errorf("unexpected API response: %d records in rrset", len(currentIP[0].Records))}
	}
	content := currentIP[0].Records[0].Content
	if content == nil {
		return false, &RecordUpdateError{Record: rrset.Name, Zone: rrset.Zone, Err: fmt.Errorf("dns record returned nil content")}
	}
	if *content == liveIP {
		slog.Info("live IP unchanged", "record", rrset.Name, "zone", rrset.Zone, "ip", liveIP)
		return false, nil
	}
	err = recordsClient.Change(ctx, rrset.Zone, rrset.Name, powerdns.RRTypeA, 60, []string{liveIP})
	if err != nil {
		return false, &RecordUpdateError{Record: rrset.Name, Zone: rrset.Zone, Err: fmt.Errorf("change dns record: %w", err)}
	}
	slog.Info("DNS record updated", "record", rrset.Name, "zone", rrset.Zone, "old_ip", *content, "new_ip", liveIP)
	return true, nil
}
