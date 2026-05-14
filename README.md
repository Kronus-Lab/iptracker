# iptracker

A DNS A record updater that monitors your public IPv4 address and keeps PowerDNS records in sync. When your IP changes, iptracker updates the configured records and can notify you via Discord or ntfy.

## Usage

### One-shot mode

Check and update once, then exit:

```sh
iptracker \
  --pdns_apikey=your-api-key \
  --pdns_url=http://powerdns:8081 \
  --rrset=myhost.example.com,example.com. \
  --rrset=otherhost.example.com,example.com.
```

### Daemon mode

Run continuously, checking at a regular interval:

```sh
iptracker \
  --pdns_apikey=your-api-key \
  --pdns_url=http://powerdns:8081 \
  --rrset=myhost.example.com,example.com. \
  --background \
  --interval=5m
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--pdns_apikey` | PowerDNS API key (required) | |
| `--pdns_url` | PowerDNS API URL (required) | |
| `--rrset` / `-r` | Comma-separated `record,zone` tuple; repeatable (required) | |
| `--background` / `-b` | Run as a continuous daemon | `false` |
| `--interval` / `-i` | Check interval in daemon mode | `5m` |
| `--discord` / `-d` | Discord webhook URL | |
| `--ntfy` / `-n` | ntfy webhook URL | |

Record names must be fully qualified (e.g. `myhost.example.com.` with trailing dot).

### Docker Compose

```yaml
services:
  iptracker:
    build: .
    command: >
      --pdns_apikey=your-api-key
      --pdns_url=http://powerdns:8081
      --rrset=myhost.example.com.,example.com.
      --background
      --interval=5m
      --ntfy=http://ntfy/example
    restart: unless-stopped
```

## Development

### Prerequisites

- Go 1.26+
- Docker (for integration tests)

### Unit tests

```sh
make test
```

### Integration tests

Integration tests require PowerDNS and ntfy running in Docker:

```sh
make test-integration
```

This starts the containers, runs the tests, and leaves the containers running for inspection. Tear down when done:

```sh
make clean-integration
```

To run a specific integration test:

```sh
docker compose -f docker-compose.integration.yml up -d --wait
go test -tags=integration -timeout 120s -v -run=TestIntegration_PowerDNS_UpdateRecord ./...
make clean-integration
```

### Test coverage

```sh
make coverage
```

### Build

```sh
go build -o iptracker .
```

### Docker build

```sh
docker build -t iptracker .
```

## Project structure

| File | Purpose |
|------|---------|
| `main.go` | CLI flags, validation, daemon loop, signal handling |
| `ip.go` | Public IP detection via ifconfig.me |
| `update.go` | Single DNS record update logic |
| `cycle.go` | Orchestrates parallel updates and notifications; webhook URL validation |
| `notify.go` | Discord and ntfy notification senders |
| `client.go` | `RecordsClient` interface and PowerDNS implementation |
| `rrset.go` | `RRSet` type and `pflag.Value` implementation |
| `errors.go` | `IPCheckError`, `RecordUpdateError`, `NotificationError` |
| `transport.go` | HTTP transport that sets curl User-Agent |

## Error types

All errors implement `Unwrap()` for use with `errors.Is` / `errors.As`:

- `IPCheckError` — failures reaching or parsing the IP check service
- `RecordUpdateError` — DNS record fetch/update failures (carries `Record` and `Zone`)
- `NotificationError` — Discord/ntfy delivery failures (carries `Service`)