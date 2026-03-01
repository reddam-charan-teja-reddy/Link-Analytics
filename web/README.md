# FlowLinks Frontend

Frontend dashboard for FlowLinks.

## Stack

- React
- Vite
- Bun
- React Router
- Recharts
- React Hot Toast

## Features

- auth (Google Sign-In)
- link creation and management
- source-link and group workflows
- analytics dashboards (summary, trend, breakdown, recent)
- light/dark theme support

## Environment

Create local env file:

```bash
cp .env.example .env
```

Variable:

- `VITE_API_BASE_URL`
  - empty for same-origin calls through Nginx proxy (default)
  - set explicitly for standalone frontend-to-backend development
- `VITE_GOOGLE_CLIENT_ID`
  - required for Google Sign-In button rendering

## Development

```bash
bun install
bun run dev
```

Default dev URL:

- `http://localhost:5173`

## Production build

```bash
bun run build
bun run preview
```

## Notes

- API wrapper uses cookie-based auth with automatic refresh.
- Request timeout is 15s by default in `src/lib/api.js`.
- Route-level chunks are split by Vite build config for better initial load.
