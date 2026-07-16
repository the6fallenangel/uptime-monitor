# uptime-monitor

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)
![CI](https://github.com/the6fallenangel/uptime-monitor/actions/workflows/ci.yml/badge.svg)

A concurrent, multi-user website/API uptime monitoring service written in Go.

Users register URLs to monitor at their own configured intervals; a worker-pool scheduler checks them concurrently, records status and response times, and sends an alert when a monitor's status changes.

## Features

- **User accounts** — signup/login with bcrypt password hashing and JWT-based sessions delivered via httpOnly cookies
- **Per-user data isolation** — every monitor is scoped to its owner; ownership is enforced at the storage layer, not just the API
- **Per-monitor, user-configurable check intervals** (`"30s"`, `"5m"`, etc.)
- **Concurrent checking via a bounded worker pool** — many monitors ticking at once doesn't mean unbounded simultaneous network requests
- **Dynamic registration** — monitors added or removed via the API take effect immediately, no restart required
- **Status-change alerting** via a pluggable `Notifier` interface — structured logs and email (SMTP) implementations included
- **Postgres-backed persistence** with cascading deletes and indexed check history
- **REST API**: signup/login/logout, create/list/delete monitors, view check history
- Graceful shutdown — in-flight checks and HTTP requests finish cleanly on exit
- Tests run against a real Postgres instance, each isolated in its own schema
- CI via GitHub Actions, including a Postgres service container

## Project layout

```
cmd/
  monitor/         entry point — wires config, storage, scheduler, notifier, and API together
internal/
  models/          User, Monitor, and Check domain types
  auth/            JWT issuing and verification
  checker/         performs a single HTTP check against a monitor
  scheduler/       worker pool, per-monitor ticking, dynamic add/remove, status-transition detection
  notifier/        Notifier interface + log and email implementations
  storage/         Postgres persistence layer, all monitor queries scoped by owner
  api/             REST API handlers, routes, and auth middleware
  config/          .env configuration loading
```

## How it works

```
monitor A ──ticker──┐
monitor B ──ticker──┼──▶ [ jobs channel ] ──▶ worker pool ──▶ storage
monitor C ──ticker──┘                              │
                                                                      ▼
                                          status changed? ──▶ Notifier
```

Each monitor runs its own ticker at its configured interval; ticks are handed to a fixed pool of worker goroutines that perform the check, persist the result, and compare it against the monitor's last known status. A notification only fires on an actual transition (e.g. up → down), not on every routine check.

## Authentication

Sessions are JWTs stored in httpOnly cookies — never exposed to client-side JavaScript, mitigating token theft via XSS. All monitor and check endpoints require a valid session and are scoped to the authenticated user; every storage query filters by owner, so one user can never read or modify another's data by guessing an ID.

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
JWT_SECRET=the6fallenangels-says-you-should-change-this

SMTP_HOST=
SMTP_PORT=
SMTP_USER=
SMTP_PASS=
ALERT_FROM=
```

If SMTP settings are left blank, status-change alerts fall back to structured log output instead of email.

### Run

```bash
go run ./cmd/monitor
```

### API

```bash
curl -c cookies.txt -X POST localhost:8080/signup \
  -d '{"name":"Ali","email":"ali@example.com","password":"supersecret123"}'

curl -b cookies.txt -X POST localhost:8080/monitors \
  -d '{"name":"Example","url":"https://example.com","interval":"30s"}'

curl -b cookies.txt localhost:8080/monitors
curl -b cookies.txt localhost:8080/monitors/1
curl -b cookies.txt localhost:8080/monitors/1/checks
curl -b cookies.txt -X DELETE localhost:8080/monitors/1
curl -b cookies.txt -X POST localhost:8080/logout
```

## Testing

```bash
docker compose up -d
go test ./... -v
```

Each test run gets its own Postgres schema, created and dropped automatically. Storage tests cover both standard CRUD and cross-user isolation (confirming one user cannot read, list, or delete another user's monitors).

## Status

Core functionality is complete: multi-user auth, ownership-scoped storage, concurrent scheduling, persistence, alerting, and a REST API. A frontend and additional hardening (CORS, expanded test coverage for auth/notifier packages) are planned next.
