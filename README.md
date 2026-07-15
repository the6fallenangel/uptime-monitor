# uptime-monitor

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)
![CI](https://github.com/the6fallenangel/uptime-monitor/actions/workflows/ci.yml/badge.svg)

A concurrent website/API uptime monitoring service written in Go.

Register URLs to monitor at user-defined intervals; a worker-pool scheduler checks them concurrently, records status and response times, and exposes everything over a REST API.

## Features

- **Per-monitor, user-configurable check intervals** (`"30s"`, `"5m"`, etc.) — not hardcoded
- **Concurrent checking via a bounded worker pool**, so many monitors ticking at once doesn't mean unbounded simultaneous network requests
- **Dynamic registration** — monitors added or removed via the API take effect immediately, no restart required
- **Postgres-backed persistence** with cascading deletes and indexed check history
- **REST API**: create/list/delete monitors, view check history, filter by result count
- Graceful shutdown — in-flight checks and HTTP requests finish cleanly on exit
- Tests run against a real Postgres instance, each isolated in its own schema
- CI via GitHub Actions, including a Postgres service container

## Project layout

```
cmd/
  monitor/         entry point — wires config, storage, scheduler, and API together
internal/
  models/          Monitor and Check domain types
  checker/         performs a single HTTP check against a monitor
  scheduler/       worker pool + per-monitor ticking, with dynamic add/remove
  storage/         Postgres persistence layer
  api/             REST API handlers and routes
  config/          .env configuration loading
```

## How it works

```
monitor A ──ticker──┐
monitor B ──ticker──┼──▶ [ jobs channel ] ──▶ worker pool ──▶ storage
monitor C ──ticker──┘
```

Each monitor runs its own ticker at its configured interval; ticks are handed off to a fixed pool of worker goroutines that perform the actual HTTP check and persist the result. This decouples "how many monitors exist" from "how much concurrent network traffic is generated" — 500 monitors ticking at once doesn't mean 500 simultaneous requests.

## Usage

### Run Postgres

```bash
docker compose up -d
```

### Configure

```bash
cp .env.example .env
```

```env
DATABASE_URL=postgres://postgres:password@localhost:5432/uptime_monitor?sslmode=disable
PORT=8080
```

### Run

```bash
go run ./cmd/monitor
```

### API

```bash
curl -X POST localhost:8080/monitors \
  -d '{"name":"Example","url":"https://example.com","interval":"30s"}'

curl localhost:8080/monitors
curl localhost:8080/monitors/1
curl localhost:8080/monitors/1/checks
curl -X DELETE localhost:8080/monitors/1
```

## Testing

```bash
docker compose up -d
go test ./... -v
```

Each test run gets its own Postgres schema, created and dropped automatically, so tests never interfere with each other or with real data.

## Status

Core functionality — scheduling, concurrent checking, persistence, REST API, dynamic monitor management — is complete and tested. Built primarily to work with Go's concurrency primitives (goroutines, channels, worker pools, context cancellation) in a genuinely useful context rather than a toy exercise.
