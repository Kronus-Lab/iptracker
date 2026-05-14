# Agent Guidelines

## Project overview

iptracker is a Go CLI that monitors your public IPv4 address and keeps PowerDNS A records in sync. It notifies via Discord or ntfy when the IP changes.

## Build and run

```sh
go build -o iptracker .
```

Run with required flags:

```sh
./iptracker --pdns_apikey=key --pdns_url=http://host:8081 --rrset=name.,zone.
```

## Testing

### Unit tests

```sh
go test -race -v ./...
```

### Integration tests

Requires Docker. Starts PowerDNS and ntfy containers:

```sh
make test-integration
# Tear down when done:
make clean-integration
```

### Both

```sh
make test && make test-integration
```

## Code style

- All code is in `package main` — single binary, no sub-packages.
- No comments unless explicitly requested. Function and type names should be self-documenting.
- Error types (`IPCheckError`, `RecordUpdateError`, `NotificationError`) are in `errors.go` and implement `Unwrap()`. Always wrap errors with these types or `fmt.Errorf` with `%w` when adding context.
- Functions return `(result, error)` — no global mutable state or in-band error signaling via atomics.
- All external collaborators (DNS client, HTTP) are behind interfaces for testability (`RecordsClient` in `client.go`).
- Notification and IP check functions return errors; callers decide whether to log or abort.

## File organization

| File | Responsibility |
|------|---------------|
| `main.go` | CLI flags, validation, daemon loop, signal handling |
| `ip.go` | Public IPv4 detection and validation |
| `update.go` | Single DNS record update logic |
| `cycle.go` | Parallel update orchestration, notifications, webhook URL validation |
| `notify.go` | Discord and ntfy HTTP notification senders |
| `client.go` | `RecordsClient` interface and PowerDNS implementation |
| `rrset.go` | `RRSet` struct and `pflag.Value` parsing |
| `errors.go` | Typed error structs with `Unwrap()` |
| `transport.go` | HTTP RoundTripper that sets curl User-Agent |
| `main_test.go` | Unit tests (mock-based, no external dependencies) |
| `integration_test.go` | Integration tests (require PowerDNS + ntfy in Docker) |

## Key conventions

- Record names in `--rrset` must be fully qualified with trailing dot (e.g. `myhost.example.com.`). The go-powerdns library calls `makeDomainCanonical` which appends a dot — a bare name like `myhost` becomes `myhost.` which PowerDNS rejects as "out of zone".
- `updateRecord` returns `(bool, error)` where `bool` indicates whether the record was actually changed (IP differed). This drives notification logic.
- `runCycle` uses local `sync.WaitGroup` and `atomic.Bool` per invocation — no package-level globals.
- Daemon mode uses `time.NewTicker` for the interval loop. Signal handling via `signal.NotifyContext` enables graceful shutdown on SIGINT/SIGTERM.
- Webhook URLs are validated at startup in `main()` via `validateWebhookURL` — must use `http` or `https` scheme and have a host.
- The IP check response from ifconfig.me is capped at 45 bytes via `io.LimitReader` and validated as IPv4 via `net.ParseIP`.

## Making changes

When modifying function signatures that appear in both `main_test.go` and `integration_test.go`, update both files. The `MockRecordsClient` in `main_test.go` must satisfy the `RecordsClient` interface.

Integration test constants:
- `testRecord` uses the FQDN form `myhost.test.example.net.` (not bare `myhost`).
- `seedZone` is idempotent — it treats HTTP 409 Conflict as success.
- The `MultipleRRSets` test creates separate zones and uses zone-specific FQDN record names.

Run `go vet ./...` and `go test ./...` after every change. Format with `gofmt -w .`.