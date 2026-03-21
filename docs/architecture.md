# Architecture Overview

This document describes the runtime architecture and request lifecycle for FlowLinks.

## Components

Backend:

- Gin HTTP server (`cmd/server/main.go`)
- service layer (`internal/service`)
- repositories (`internal/repository/postgres`, `internal/repository/redis`)
- async click worker (`internal/worker/click_worker.go`)

Frontend:

- React app (`web/src`)
- API client and page-level service wrappers (`web/src/lib`)

Data stores:

- PostgreSQL (source of truth)
- Redis (redirect path cache)

## Route model

Public routes:

- `GET /health`
- `GET /:hash`

Auth routes:

- `/auth/google`
- `/auth/refresh`
- `/auth/me`

Deprecated endpoints still present for explicit messaging:

- `/auth/register` (returns 410)
- `/auth/login` (returns 410)

Authenticated API routes:

- `/api/links*`
- `/api/groups*`
- `/api/sources/batch`
- `/api/links/:linkId/analytics*`

## Redirect path (latency-sensitive)

1. Client requests `/:hash`.
2. Redirect service checks Redis key `link:<hash>`.
3. On miss, backend resolves hash in Postgres.
4. Response returns `302` immediately with destination URL.
5. Click metadata is queued asynchronously.

Current design trade-off:

- redirect latency is prioritized
- when worker channel is saturated, overflow events are persisted to Redis stream

## Analytics ingestion path (throughput-oriented)

1. Redirect handler creates click event payload.
2. Worker queue buffers events in memory.
3. Overflow events are temporarily written to Redis stream when queue is full.
4. Worker drains queue + overflow stream, resolves GeoIP, and batches inserts.
5. Batches flush every 100 events or 500ms.

Failure behavior:

- enqueue is non-blocking; full channel falls back to Redis overflow stream
- overflow entries are acknowledged only after successful DB persistence
- flush failures use bounded backoff before retrying
- events may still be dropped if both in-memory queue and Redis overflow writes fail

## Link model

- base link: one hash maps to one destination URL
- source link: many per base link (`source_name`, own hash)
- group link: many-to-many relation through `link_groups`
- click event: references `link_id` and optional `source_link_id`

## Security and control plane

- JWT auth middleware for `/api/*`
- refresh-token endpoint for rolling session renewal (`/auth/refresh`)
- CORS middleware with configured origins
- rate limiting middleware by scope (`auth`, `api`, `redirect`)
- request ID + request logging middleware

Trusted IP behavior:

- Gin trusted proxies are configured via `TRUSTED_PROXIES`.

Rate limiting behavior:

- Redis-backed distributed limiting when Redis is available.
- Automatic in-memory per-instance fallback during Redis outages.

## Caching

- Redis cache key format: `link:<hash>`
- cache TTL: configurable via `LINK_CACHE_TTL_SECONDS` (default 24h)
- invalidation on link disable/delete and source delete

## GeoIP enrichment behavior

- GeoIP lookups are cached (positive + short negative cache)
- resolver uses bounded retries with incremental backoff
- repeated upstream failures trigger a temporary circuit-open window
- fallback values are `Unknown` when upstream cannot be resolved safely

## Backend layering

Handler layer:

- HTTP validation and response mapping

Service layer:

- ownership checks
- business rules
- composition across repositories

Repository layer:

- SQL and Redis operations only

Worker layer:

- async click persistence

## Frontend architecture

- API wrapper (`web/src/lib/api.js`) centralizes:
  - base URL
  - auth header injection
  - timeout handling
  - normalized errors
- service wrappers (`web/src/lib/services.js`) map endpoint families
- page components own UI state and call services

## Observability signals available now

- request logs with request ID
- health endpoint
- docker logs for backend/frontend

Recommended next improvements:

- structured logs (JSON)
- metrics endpoint
- tracing across redirect and worker flows
