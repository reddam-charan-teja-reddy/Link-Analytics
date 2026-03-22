# Full Cloud Deployment Guide (Beginner-Friendly)

This guide deploys this project in a production-ready way with:

- Backend on Google Cloud Run
- Frontend on Google Cloud Run (or static hosting option)
- PostgreSQL and Redis as managed services
- Google OAuth Sign-In
- Secrets in Secret Manager
- Custom domain and HTTPS

It assumes you are starting from zero.

## 1) What You Will Build

By the end, you will have:

1. A backend service URL, for example `https://api-xxxxx-uc.a.run.app`
2. A frontend service URL, for example `https://app-xxxxx-uc.a.run.app`
3. Working Google Sign-In
4. Database and Redis connected
5. Health endpoints available:
   - `/health/live`
   - `/health/ready`

## 2) Accounts and Tools You Need

1. Google Cloud account with billing enabled
2. A domain name (optional, but recommended)
3. Local tools installed:
   - `git`
   - `docker`
   - `gcloud` CLI

Install `gcloud`:

```bash
curl https://sdk.cloud.google.com | bash
exec -l $SHELL
gcloud init
```

## 3) Create Google Cloud Project

1. Open Google Cloud Console.
2. Create a new project (example: `url-shortener-prod`).
3. Note your:
   - Project ID
   - Project Number

Set local shell variables (replace values):

```bash
export PROJECT_ID="your-project-id"
export REGION="us-central1"
gcloud config set project "$PROJECT_ID"
```

## 4) Enable Required APIs

Enable APIs used in this deployment:

```bash
gcloud services enable run.googleapis.com \
  artifactregistry.googleapis.com \
  cloudbuild.googleapis.com \
  secretmanager.googleapis.com
```

## 5) IAM Roles You Need

According to Cloud Run docs, deployers typically need:

- `roles/run.developer`
- `roles/iam.serviceAccountUser`
- `roles/artifactregistry.reader` (for pulling images)

If managing secrets in deployment, you also need permissions to configure them. For secret access at runtime, the runtime service account needs:

- `roles/secretmanager.secretAccessor`

## 6) Provision PostgreSQL and Redis (Cheapest Reliable Path)

For low cost, simple operations:

1. PostgreSQL: use Neon free or low-tier paid
2. Redis: use Redis Cloud free 30MB (or upgrade when needed)

Collect these values:

- `DATABASE_URL` (Postgres connection string with TLS enabled)
- `REDIS_URL` (Redis connection URL)

Important:

- In production, this app rejects `sslmode=disable` for Postgres.
- Keep credentials out of git and out of plain env files.

## 7) Create Secrets in Secret Manager

Create secrets for sensitive values:

```bash
printf '%s' 'replace-with-strong-jwt-secret-min-32-chars' | gcloud secrets create jwt-secret --data-file=-
printf '%s' 'postgres://...' | gcloud secrets create database-url --data-file=-
printf '%s' 'redis://...' | gcloud secrets create redis-url --data-file=-
printf '%s' 'your-google-client-id.apps.googleusercontent.com' | gcloud secrets create google-client-id --data-file=-
```

If secret already exists, add new version:

```bash
printf '%s' 'new-value' | gcloud secrets versions add jwt-secret --data-file=-
```

## 8) Create Artifact Registry Repository

```bash
gcloud artifacts repositories create url-shortener \
  --repository-format=docker \
  --location="$REGION" \
  --description="URL shortener images"
```

Configure Docker auth for Artifact Registry:

```bash
gcloud auth configure-docker "$REGION-docker.pkg.dev"
```

## 9) Build and Push Backend Image

From repo root:

```bash
export BACKEND_IMAGE="$REGION-docker.pkg.dev/$PROJECT_ID/url-shortener/backend:latest"
docker build -t "$BACKEND_IMAGE" -f Dockerfile .
docker push "$BACKEND_IMAGE"
```

## 10) Build and Push Frontend Image

```bash
export FRONTEND_IMAGE="$REGION-docker.pkg.dev/$PROJECT_ID/url-shortener/frontend:latest"
docker build -t "$FRONTEND_IMAGE" -f Dockerfile.frontend ./web
docker push "$FRONTEND_IMAGE"
```

## 11) Create Runtime Service Account

```bash
gcloud iam service-accounts create url-shortener-runtime \
  --display-name="URL Shortener Runtime"
```

Grant secret read to runtime SA:

```bash
export RUNTIME_SA="url-shortener-runtime@$PROJECT_ID.iam.gserviceaccount.com"

for s in jwt-secret database-url redis-url google-client-id; do
  gcloud secrets add-iam-policy-binding "$s" \
	 --member="serviceAccount:$RUNTIME_SA" \
	 --role="roles/secretmanager.secretAccessor"
done
```

## 12) Deploy Backend to Cloud Run

This app listens on `PORT` and Cloud Run injects `PORT` automatically. Do not set `PORT` manually unless needed.

Deploy backend service:

```bash
gcloud run deploy url-shortener-api \
  --image "$BACKEND_IMAGE" \
  --region "$REGION" \
  --platform managed \
  --allow-unauthenticated \
  --service-account "$RUNTIME_SA" \
  --min-instances 0 \
  --max-instances 10 \
  --set-env-vars APP_ENV=production,RUN_MIGRATIONS=false,REFRESH_COOKIE_SECURE=true,REFRESH_COOKIE_DOMAIN= \
  --set-secrets JWT_SECRET=jwt-secret:latest,DATABASE_URL=database-url:latest,REDIS_URL=redis-url:latest,GOOGLE_CLIENT_ID=google-client-id:latest
```

After deploy, get backend URL:

```bash
gcloud run services describe url-shortener-api --region "$REGION" --format='value(status.url)'
```

Save it:

```bash
export API_URL="https://your-api-url.run.app"
```

## 13) Run Database Migrations Safely

Use one controlled run. Temporarily deploy a revision with `RUN_MIGRATIONS=true`, wait for startup success, then deploy again with `RUN_MIGRATIONS=false`.

```bash
gcloud run deploy url-shortener-api \
  --image "$BACKEND_IMAGE" \
  --region "$REGION" \
  --platform managed \
  --allow-unauthenticated \
  --service-account "$RUNTIME_SA" \
  --set-env-vars APP_ENV=production,RUN_MIGRATIONS=true,REFRESH_COOKIE_SECURE=true,REFRESH_COOKIE_DOMAIN= \
  --set-secrets JWT_SECRET=jwt-secret:latest,DATABASE_URL=database-url:latest,REDIS_URL=redis-url:latest,GOOGLE_CLIENT_ID=google-client-id:latest
```

Then switch back to `RUN_MIGRATIONS=false` using the deploy command from section 12.

## 14) Deploy Frontend to Cloud Run

Deploy frontend service and point it at backend URL:

```bash
gcloud run deploy url-shortener-web \
  --image "$FRONTEND_IMAGE" \
  --region "$REGION" \
  --platform managed \
  --allow-unauthenticated \
  --min-instances 0 \
  --max-instances 5 \
  --set-env-vars VITE_API_BASE_URL="$API_URL",VITE_GOOGLE_CLIENT_ID="your-google-client-id.apps.googleusercontent.com"
```

Get frontend URL:

```bash
gcloud run services describe url-shortener-web --region "$REGION" --format='value(status.url)'
```

## 15) Configure Google OAuth (Required)

From Google Identity docs, you need a web client ID and authorized origins.

1. Open Google Cloud Console -> Google Auth Platform -> Clients
2. Create OAuth Client ID, type: Web application
3. Add Authorized JavaScript origins:
   - Frontend Cloud Run URL
   - Local dev origins (`http://localhost:5173`)
4. Add OAuth Consent Screen branding details
5. Copy client ID and update:
   - backend secret `google-client-id`
   - frontend env `VITE_GOOGLE_CLIENT_ID`

If you use a custom domain, add it to authorized origins too.

## 16) Configure CORS and Cookies Correctly

Set backend `ALLOWED_ORIGINS` to your frontend URL(s). Example:

```bash
gcloud run services update url-shortener-api \
  --region "$REGION" \
  --update-env-vars ALLOWED_ORIGINS="https://your-frontend.run.app"
```

Cookie guidance:

- Keep `REFRESH_COOKIE_SECURE=true` in production
- Use `REFRESH_COOKIE_DOMAIN` only when needed for parent-domain sharing

## 17) Verify Deployment

Backend checks:

```bash
curl -s "$API_URL/health/live"
curl -s "$API_URL/health/ready"
```

Manual functional checks:

1. Open frontend URL
2. Click Sign in with Google
3. Create a short link
4. Open short URL and confirm redirect works

## 18) Custom Domain (Production Recommendation)

Cloud Run docs recommend using a global external Application Load Balancer for production custom domains.

You can use Cloud Run native domain mapping, but docs mark it limited/preview and not recommended for production-critical usage.

Recommended approach:

1. Put Cloud Run service behind Global External Application Load Balancer
2. Attach managed certificate
3. Point DNS to LB

If you still use Cloud Run domain mapping:

1. Go to Cloud Run Domain Mappings page
2. Add mapping
3. Verify domain ownership
4. Add provided DNS records at registrar
5. Wait for certificate issuance (can take minutes to hours)

## 19) Logging, Monitoring, and Alerts

1. Cloud Run -> Service -> Logs: verify request logs and startup logs
2. Create uptime check for `/health/ready`
3. Add alert policy:
   - high 5xx rate
   - latency spikes
   - instance crash loops

## 20) Cost Controls

To keep costs low while reliable:

1. Keep min instances at `0` initially
2. Set max instances to a safe cap
3. Use Neon + Redis Cloud free tiers until traffic grows
4. Monitor egress and database usage monthly
5. Scale paid plans only after real usage pressure

## 21) Safe Rollback Procedure

Cloud Run revisions are immutable. Rollback is easy:

1. Open Cloud Run service
2. Revisions tab
3. Select previous healthy revision
4. Route 100% traffic to that revision

CLI example:

```bash
gcloud run services update-traffic url-shortener-api \
  --region "$REGION" \
  --to-revisions REVISION_NAME=100
```

## 22) Production Checklist

Before going live, confirm all are true:

1. `APP_ENV=production`
2. Postgres TLS enabled (`sslmode=disable` not used)
3. Strong JWT secret (32+ chars)
4. `GOOGLE_CLIENT_ID` set in backend
5. `VITE_GOOGLE_CLIENT_ID` set in frontend
6. `ALLOWED_ORIGINS` only includes your real domains
7. `/health/live` and `/health/ready` both working
8. Google sign-in works on real domain
9. Logs and alerts configured

## 23) Common Issues and Fixes

1. Google popup opens but login fails:
   - Check authorized origin exactly matches frontend domain/protocol
   - Confirm same client ID is used in backend and frontend
2. 401 after login:
   - Check cookies are set and allowed by browser
   - Recheck `ALLOWED_ORIGINS` and secure cookie settings
3. Backend start fails on production:
   - Missing required env (`GOOGLE_CLIENT_ID`, `JWT_SECRET`, `DATABASE_URL`, etc.)
4. DB connection errors:
   - Ensure provider IP/rules allow Cloud Run egress
   - Verify connection string and TLS settings

## 24) Optional: Fully on Google Cloud Alternative

If you want all-GCP stack:

1. Cloud SQL for PostgreSQL
2. Memorystore for Redis
3. Cloud Run for backend/frontend

This is operationally clean but often costs more than Neon + Redis Cloud at small scale.
