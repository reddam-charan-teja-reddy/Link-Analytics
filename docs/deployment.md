# Deployment Guide (Local + Production Baseline)

This guide covers:

- local deployment with Docker Compose
- production baseline decisions
- troubleshooting and operational checks

For cloud-specific runbooks, see `cloudDeployment.md`.

## Local deployment (Docker Compose)

### 1. Configure environment

```bash
cp .env.example .env
cp web/.env.example web/.env
```

Review at minimum:

- `POSTGRES_PASSWORD`
- `JWT_SECRET`
- `APP_ENV`
- `BASE_URL`
- `ALLOWED_ORIGINS`
- `TRUSTED_PROXIES`

### 2. Build and start

```bash
docker compose up --build -d
```

### 3. Validate

```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8080/health/live
curl -s http://localhost:8080/health/ready
curl -I http://localhost:3000
```

### 4. Logs

```bash
docker compose logs -f backend frontend
```

### 5. Stop

```bash
docker compose down
```

## Production baseline architecture

Recommended split:

- frontend static assets at edge/CDN
- backend API in containers
- managed PostgreSQL
- managed Redis

Why:

- simpler scaling and patching
- better reliability than self-managed DB/Redis on one VM
- lower operational burden

## Environment variables (backend)

Required:

- `DATABASE_URL`
- `REDIS_URL`
- `JWT_SECRET`
- `BASE_URL`
- `ALLOWED_ORIGINS`

Recommended:

- `APP_ENV=production`

Optional (with defaults):

- `PORT` (default `8080`)
- `RUN_MIGRATIONS`
- `MIGRATIONS_DIR`
- `AUTH_RATE_LIMIT_PER_MINUTE`
- `API_RATE_LIMIT_PER_MINUTE`
- `REDIRECT_RATE_LIMIT_PER_MINUTE`
- `LINK_CACHE_TTL_SECONDS` (default `86400`)
- `REFRESH_COOKIE_SECURE` (default `true` in production)
- `REFRESH_COOKIE_DOMAIN` (default empty)

Frontend:

- `VITE_API_BASE_URL`
- `VITE_GOOGLE_CLIENT_ID`

## Operational hardening checklist

- set strong `JWT_SECRET` (do not use defaults)
- enable TLS everywhere (edge and backend origin)
- restrict CORS (`ALLOWED_ORIGINS`) to known frontends
- run database with backups enabled
- run Redis with auth/network isolation
- ensure clock sync (JWT expiry behavior depends on system time)
- monitor container restart loops
- verify logs include request IDs

## Migrations

Current behavior:

- backend container runs with `RUN_MIGRATIONS=true` in compose

Production recommendation:

- run migrations as explicit release step before app rollout
- avoid concurrent migration execution from multiple replicas

## Health and smoke checks

After deployment:

1. `GET /health/live` returns `200`.
2. `GET /health/ready` returns `200` and component status payload.
3. `GET /health` (alias of readiness) returns `200` for healthy/degraded app state.
4. Google Sign-In works.
5. Create a link and open short hash.
6. Analytics summary shows click after a short delay.
7. Group and source-link flows behave as expected.

## Known production caveats

- click queue persists overflow events to Redis stream, but events may still be dropped if both queue and Redis are unavailable
- overflow stream draining runs in a single worker to prevent duplicate ingestion; entries are acknowledged after DB persistence
- rate limiting is Redis-backed when Redis is healthy; fallback is in-memory per instance during Redis outages
- GeoIP enrichment gracefully degrades to `Unknown` during upstream failures; analytics remain functional but less location-precise during outages
- source-filter mode in group detail currently shows source-tag view (`?src=...`) instead of source-hash expansion

These caveats do not block deployment, but should be tracked as engineering debt.

## Common issues

### Frontend loads but API calls fail

Check:

- `web/nginx.conf` proxy routes
- backend container health
- CORS origins
- `GOOGLE_CLIENT_ID` and `VITE_GOOGLE_CLIENT_ID` values are set

### Redirect works but analytics looks sparse

Likely causes:

- queue drops during burst traffic
- worker flush errors (check backend logs)
- low traffic window with delayed flush timing

### 429 responses appear unexpectedly

Tune:

- `AUTH_RATE_LIMIT_PER_MINUTE`
- `API_RATE_LIMIT_PER_MINUTE`
- `REDIRECT_RATE_LIMIT_PER_MINUTE`

Remember these are per-instance limits in current design.

When Redis is healthy, distributed limits are enforced across instances.

## Audit status

- latest consolidated audit and fix verification: refer to current CI checks and local validation commands in this guide
