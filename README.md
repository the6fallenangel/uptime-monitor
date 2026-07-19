# uptime-monitor

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)
![CI](https://github.com/the6fallenangel/uptime-monitor/actions/workflows/ci.yml/badge.svg)

A concurrent, multi-user website/API uptime monitoring service written in Go.

Users register URLs to monitor at their own configured intervals; a worker-pool scheduler checks them concurrently, records status and response times, and sends an alert when a monitor's status changes.

## Features

- **User accounts** — signup/login with bcrypt password hashing and JWT-based sessions delivered via httpOnly cookies
- **Account management** — fetch the current session's user (`/me`), update display name, change password (current-password confirmation required)
- **Per-user data isolation** — every monitor is scoped to its owner; ownership is enforced at the storage layer, not just the API
- **Per-monitor, user-configurable check intervals** (`"30s"`, `"5m"`, etc.), editable after creation — the scheduler restarts that monitor's ticker automatically when its interval changes
- **Concurrent checking via a bounded worker pool** — many monitors ticking at once doesn't mean unbounded simultaneous network requests
- **Dynamic registration** — monitors added, updated, or removed via the API take effect immediately, no restart required
- **Self-healing scheduler** — if a monitor is deleted out from under a running scheduler (e.g. directly in the database), its ticker detects the failure and stops itself instead of erroring indefinitely
- **Paginated check history** — `page`/`limit` query params with total count and page count in the response
- **Status-change alerting** via a pluggable `Notifier` interface — structured logs and email (SMTP) implementations included
- **CORS support** with credentialed requests, ready for a separate frontend origin
- **Postgres-backed persistence** with cascading deletes and indexed check history
- **REST API**: signup/login/logout, account management, create/update/list/delete monitors, paginated check history
- Graceful shutdown — in-flight checks and HTTP requests finish cleanly on exit
- Full test suite: storage (with cross-user isolation), auth, notifier, scheduler transition logic, and API handlers — each Postgres-backed test isolated in its own schema
- CI via GitHub Actions, including a Postgres service container

## Project layout

```
cmd/
  monitor/         entry point — wires config, storage, scheduler, notifier, and API together
internal/
  models/          User, Monitor, and Check domain types
  auth/            JWT issuing and verification
  checker/         performs a single HTTP check against a monitor
  scheduler/       worker pool, per-monitor ticking, dynamic add/remove/update, status-transition detection, self-healing on missing monitors
  notifier/        Notifier interface + log and email implementations
  storage/         Postgres persistence layer, all monitor queries scoped by owner
  api/             REST API handlers, routes, auth middleware, and CORS
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

Each monitor runs its own ticker at its configured interval; ticks are handed to a fixed pool of worker goroutines that perform the check, persist the result, and compare it against the monitor's last known status. A notification only fires on an actual transition (e.g. up → down), not on every routine check. Updating a monitor's interval restarts its ticker; if a check fails to save because the monitor no longer exists, the scheduler stops ticking it rather than retrying forever.

## Authentication

Sessions are JWTs stored in httpOnly cookies — never exposed to client-side JavaScript, mitigating token theft via XSS. All monitor, check, and account endpoints require a valid session and are scoped to the authenticated user; every storage query filters by owner, so one user can never read or modify another's data by guessing an ID.

The session cookie's `Secure` flag is environment-aware: disabled in development (so it works over plain HTTP with a local frontend dev server), enabled in production (HTTPS only).

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
ENVIRONMENT=development
DATABASE_URL=postgres://postgres:password@localhost:5432/uptime_monitor?sslmode=disable
PORT=8080
JWT_SECRET=change-this-to-a-long-random-string
FRONTEND_ORIGIN=http://localhost:3000

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

curl -b cookies.txt localhost:8080/me
curl -b cookies.txt -X PATCH localhost:8080/me/name -d '{"name":"New Name"}'
curl -b cookies.txt -X POST localhost:8080/me/password \
  -d '{"currentPassword":"supersecret123","newPassword":"newpassword123"}'

curl -b cookies.txt -X POST localhost:8080/monitors \
  -d '{"name":"Example","url":"https://example.com","interval":"30s"}'

curl -b cookies.txt localhost:8080/monitors
curl -b cookies.txt localhost:8080/monitors/1
curl -b cookies.txt -X PATCH localhost:8080/monitors/1 \
  -d '{"name":"Renamed","interval":"5m"}'
curl -b cookies.txt "localhost:8080/monitors/1/checks?page=1&limit=20"
curl -b cookies.txt -X DELETE localhost:8080/monitors/1
curl -b cookies.txt -X POST localhost:8080/logout
```

## Testing

```bash
docker compose up -d
go test ./... -v
```

Every Postgres-backed test runs in its own dynamically created schema, dropped automatically afterward, so tests never interfere with each other or with real data. Coverage includes:

- Storage CRUD, updates, and cross-user isolation (one user cannot read, list, update, or delete another user's monitors)
- JWT issuing, verification, tampering, expiry, and secret mismatches
- Scheduler status-transition detection, using a mock notifier and a stub storage layer
- API-level auth flows (signup, login, duplicate email, wrong password), account management, and ownership enforcement over HTTP

## Status

Backend is feature-complete: multi-user auth, account management, ownership-scoped storage, concurrent scheduling with dynamic updates and self-healing, paginated persistence, alerting, CORS, and full test coverage across all packages. Paired with a Next.js frontend at [uptime-monitor-web](https://github.com/the6fallenangel/uptime-monitor-web).
