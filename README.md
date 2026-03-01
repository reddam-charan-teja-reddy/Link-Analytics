# FlowLinks URL Shortener

FlowLinks is a full-stack URL shortener and analytics platform.

It supports:

- user authentication
- short-link creation
- source-specific link variants
- group organization
- click analytics (summary, trends, breakdowns, recent activity)

## Repository layout

- `cmd/server` - backend bootstrap and route wiring
- `internal` - backend handlers, services, repositories, worker, config
- `pkg` - shared utilities (hashing, geoip, bot/user-agent detection)
- `web` - React frontend (Vite + Bun)
- `docs` - architecture, API, deployment, and cloud guides

## Architecture summary

High-level request flow:

1. User creates a base link (and optional source links).
2. Visitor opens `/:hash`.
3. Backend resolves hash via Redis or Postgres.
4. Backend returns HTTP 302 immediately.
5. Click event is queued to worker and persisted asynchronously.
6. Dashboard reads analytics endpoints for charts/tables.

## Tech stack

Backend:

- Go 1.24
- Gin
- PostgreSQL
- Redis

Frontend:

- React
- Vite
- Bun
- Recharts

Runtime:

- Docker + Docker Compose
- Nginx (serving frontend + reverse proxy)

## Quick start (local, Docker)

1. Copy environment files.

```bash
cp .env.example .env
cp web/.env.example web/.env
```

2. Build and run.

```bash
docker compose up --build -d
```

3. Open services.

- Frontend: `http://localhost:3000`
- Backend health: `http://localhost:8080/health`

4. Tail logs.

```bash
docker compose logs -f backend frontend
```

5. Stop stack.

```bash
docker compose down
```

## Local development without Docker

Backend:

```bash
go run ./cmd/server
```

Frontend:

```bash
cd web
bun install
bun run dev
```

## Core environment variables

Backend (`.env`):

- `DATABASE_URL`
- `REDIS_URL`
- `JWT_SECRET`
- `GOOGLE_CLIENT_ID`
- `BASE_URL`
- `PORT`
- `RUN_MIGRATIONS`
- `MIGRATIONS_DIR`
- `ALLOWED_ORIGINS`
- `AUTH_RATE_LIMIT_PER_MINUTE`
- `API_RATE_LIMIT_PER_MINUTE`
- `REDIRECT_RATE_LIMIT_PER_MINUTE`
- `LINK_CACHE_TTL_SECONDS`
- `TRUSTED_PROXIES`

Frontend (`web/.env`):

- `VITE_API_BASE_URL` (empty by default for same-origin through Nginx)
- `VITE_GOOGLE_CLIENT_ID`

## Documentation

- `docs/architecture.md` - internals, lifecycle, and data flow
- `docs/api-reference.md` - endpoint contracts and request/response details
- `docs/deployment.md` - local + production deployment notes
- `web/README.md` - frontend-specific guide

## Known limitations (current behavior)

- click events can still be lost if both in-memory queue and Redis overflow persistence are unavailable
- no persistent long-term analytics queue (current overflow durability depends on Redis availability)
- source-filter mode in group detail currently uses source-tag view (`?src=...`) rather than source-hash expansion

See docs for mitigation and hardening guidance.
