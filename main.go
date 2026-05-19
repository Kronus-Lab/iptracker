package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"
)

func main() {
	pdnsApiKey := pflag.String("pdns_apikey", "", "Set the PowerDNS API Key")
	pdnsApiUrl := pflag.String("pdns_url", "", "Set the PowerDNS API URL")
	var recordSets rrsetSlice
	pflag.VarP(&recordSets, "rrset", "r", "Provide one or more comma separated tuple of record,zone.")

	isDaemon := pflag.BoolP("background", "b", false, "Set if running as a continous background process.")
	interval := pflag.DurationP("interval", "i", 5*time.Minute, "Set the check interval when running in daemon mode.")
	ntfy := pflag.StringP("ntfy", "n", "", "Set a NTFY webhook.")
	discord := pflag.StringP("discord", "d", "", "Set a Discord webhook.")
	pflag.Parse()

	if *pdnsApiKey == "" {
		fmt.Fprintln(os.Stderr, "error: --pdns_apikey is required")
		pflag.Usage()
		os.Exit(1)
	}
	if *pdnsApiUrl == "" {
		fmt.Fprintln(os.Stderr, "error: --pdns_url is required")
		pflag.Usage()
		os.Exit(1)
	}
	if len(recordSets) == 0 {
		fmt.Fprintln(os.Stderr, "error: at least one --rrset is required")
		pflag.Usage()
		os.Exit(1)
	}
	if *isDaemon && *interval <= 0 {
		fmt.Fprintln(os.Stderr, "error: --interval must be positive")
		os.Exit(1)
	}
	if err := validateWebhookURL(*discord); err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid --discord: %v\n", err)
		os.Exit(1)
	}
	if err := validateWebhookURL(*ntfy); err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid --ntfy: %v\n", err)
		os.Exit(1)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = 20
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if !*isDaemon {
		runOnce(ctx, httpClient, *pdnsApiKey, *pdnsApiUrl, *discord, *ntfy, recordSets)
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	runOnce(ctx, httpClient, *pdnsApiKey, *pdnsApiUrl, *discord, *ntfy, recordSets)
	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down", "reason", ctx.Err())
			return
		case <-ticker.C:
			slog.Info("checking live IP address")
			runOnce(ctx, httpClient, *pdnsApiKey, *pdnsApiUrl, *discord, *ntfy, recordSets)
		}
	}
}

func runOnce(ctx context.Context, httpClient *http.Client, pdnsApiKey, pdnsApiUrl, discord, ntfy string, recordSets rrsetSlice) {
	liveIP, err := getLiveIP(ctx, httpClient, defaultIPCheckURL)
	if err != nil {
		slog.Warn("failed to get live IP", "error", err)
		return
	}
	slog.Info("live IP retrieved", "ip", liveIP)
	recordsClient := newRecordsClient(pdnsApiKey, pdnsApiUrl, httpClient)
	runCycle(ctx, liveIP, recordSets, recordsClient, discord, ntfy, httpClient)
}
