# API Reference

Base URL (local): `http://localhost:8080`

All `/api/*` routes require auth. Preferred mode is HttpOnly cookie auth.

Optional API clients can still pass bearer token:

- `Authorization: Bearer <token>`

## Auth endpoints

### POST `/auth/google`

Sign in or register using a Google ID token from Google Identity Services.

Request:

```json
{
  "credential": "<google-id-token>"
}
```

Success `200`:

```json
{
  "user": {
    "id": "<uuid>",
    "email": "user@example.com",
    "created_at": "2026-03-22T12:00:00Z"
  }
}
```

Response also sets HttpOnly cookies:

- `access_token` (path `/`, ~15m)
- `refresh_token` (path `/auth`, ~30d)

### POST `/auth/register`

Deprecated for security policy. Email/password registration is disabled.

Response `410`:

```json
{
  "error": "email registration is disabled; use Sign in with Google"
}
```

### POST `/auth/login`

Deprecated for security policy. Email/password login is disabled.

Response `410`:

```json
{
  "error": "email login is disabled; use Sign in with Google"
}
```

### POST `/auth/refresh`

Rotate session cookies and return current user.

Request body is optional (cookie mode is preferred). Legacy fallback body:

```json
{
  "refresh_token": "<refresh-jwt>"
}
```

Success `200`:

```json
{
  "user": {
    "id": "<uuid>",
    "email": "user@example.com",
    "created_at": "2026-03-22T12:00:00Z"
  }
}
```

Response rotates `access_token` and `refresh_token` HttpOnly cookies.

### GET `/auth/me`

Returns current authenticated user.

Success `200`:

```json
{
  "id": "<uuid>",
  "email": "user@example.com",
  "created_at": "2026-03-22T12:00:00Z"
}
```

## Link endpoints

### GET `/api/links`

List links for current user.

Query params:

- `group_id` (optional)
- `source` (optional)

Success `200`:

- array of link objects with `short_url`

### POST `/api/links`

Create base short link.

Request:

```json
{
  "original_url": "https://example.com/landing",
  "title": "Campaign landing"
}
```

Success `201`:

- created link object with `short_url`

### GET `/api/links/:linkId`

Get one link and metadata.

Success `200`:

- link object with `short_url`

### PUT `/api/links/:linkId`

Update title and/or active state.

Request:

```json
{
  "title": "Updated title",
  "is_active": true
}
```

Success `200`:

- updated link object with `short_url`

### DELETE `/api/links/:linkId`

Delete a base link.

Success `200`:

```json
{
  "message": "link deleted"
}
```

## Source link endpoints

### GET `/api/links/:linkId/sources`

List source links for a base link.

Success `200`:

- array of source-link objects with `short_url`

### POST `/api/links/:linkId/sources`

Create one source link under a base link.

Request:

```json
{
  "source_name": "linkedin"
}
```

Success `201`:

- created source-link object with `short_url`

### DELETE `/api/links/:linkId/sources/:sourceId`

Delete a source link.

Success `200`:

```json
{
  "message": "source deleted"
}
```

### POST `/api/sources/batch`

Batch-create source links in scope (`all`, `group`).

Request:

```json
{
  "source_name": "linkedin",
  "scope_type": "group",
  "scope_id": "<group-uuid>"
}
```

Success `200`:

```json
{
  "created_count": 5,
  "skipped_count": 2,
  "items": [
    {
      "id": "<uuid>",
      "link_id": "<uuid>",
      "source_name": "linkedin",
      "hash": "abc123",
      "short_url": "http://localhost:8080/abc123",
      "created_at": "2026-03-22T12:00:00Z"
    }
  ]
}
```

## Group endpoints

### GET `/api/groups`

List groups for current user.

Success `200`:

```json
[
  {
    "id": "<uuid>",
    "user_id": "<uuid>",
    "name": "Q2 hiring",
    "link_count": 12,
    "created_at": "2026-03-22T12:00:00Z"
  }
]
```

### POST `/api/groups`

Create group.

Request:

```json
{
  "name": "Hiring campaign"
}
```

### PUT `/api/groups/:groupId`

Rename group.

Request:

```json
{
  "name": "Q2 hiring"
}
```

### DELETE `/api/groups/:groupId`

Delete group.

Success `200`:

```json
{
  "message": "group deleted"
}
```

### GET `/api/links/:linkId/groups`

List groups containing this link.

### POST `/api/groups/:groupId/links`

Add link to group.

Request:

```json
{
  "link_id": "<link-uuid>"
}
```

Success `200`:

```json
{
  "message": "link added to group"
}
```

### DELETE `/api/groups/:groupId/links/:linkId`

Remove link from group.

Success `200`:

```json
{
  "message": "link removed from group"
}
```

## Analytics endpoints

Date range query params (where applicable):

- `from=YYYY-MM-DD`
- `to=YYYY-MM-DD`

### GET `/api/links/:linkId/analytics`

Summary.

Success `200`:

```json
{
  "total_clicks": 42,
  "unique_visitors": 30,
  "bot_clicks": 2,
  "last_clicked_at": "2026-03-22T14:00:00Z"
}
```

### GET `/api/links/:linkId/analytics/clicks`

Time series.

Query params:

- `granularity=day|hour` (default `day`)

Success `200`:

```json
[
  { "timestamp": "2026-03-20T00:00:00Z", "clicks": 5 },
  { "timestamp": "2026-03-21T00:00:00Z", "clicks": 8 }
]
```

### GET `/api/links/:linkId/analytics/sources`

### GET `/api/links/:linkId/analytics/referrers`

### GET `/api/links/:linkId/analytics/locations`

### GET `/api/links/:linkId/analytics/browsers`

Each returns:

```json
[
  { "label": "linkedin", "clicks": 10 },
  { "label": "github", "clicks": 6 }
]
```

### GET `/api/links/:linkId/analytics/recent`

Recent click rows.

Query params:

- `limit` (default 20, max 100)

## Redirect endpoint

### GET `/:hash`

Public endpoint.

Behavior:

- resolves base hash or source hash
- merges incoming query params into destination URL
- if `?src=...` is present, records it in click event payload
- returns `302 Found` on success
- returns `404` if hash is unknown

## Common error shape

Most errors return:

```json
{
  "error": "message"
}
```
