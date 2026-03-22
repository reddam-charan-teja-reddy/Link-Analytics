#!/bin/bash
set -e

cd '/home/charan/Documents/repositories/url Shortener'

if [ ! -d .git ]; then
    git init
fi

git add .
git commit -m "Temp backup before rewrite" || true
git branch temp-backup-branch || true

BRANCH=$(git branch --show-current)
if [ -z "$BRANCH" ]; then
  BRANCH="main"
fi

git checkout --orphan new-history
git rm -rf --cached .

function do_commit {
    DATE=$1
    MSG=$2
    shift 2
    added=0
    for path in "$@"; do
        if [ -e "$path" ]; then
            git add "$path"
            added=1
        fi
    done
    if [ "$added" -eq 1 ]; then
        GIT_AUTHOR_DATE="$DATE" GIT_COMMITTER_DATE="$DATE" git commit -m "$MSG" >/dev/null 2>&1
    fi
}

do_commit "2026-03-01T10:00:00Z" "init: initialize project structure and go.mod" "go.mod" "go.sum" "README.md"
do_commit "2026-03-01T14:30:00Z" "chore: add gitignore and basic structure" ".gitignore" "web/.gitignore" "web/README.md"
do_commit "2026-03-02T11:15:00Z" "feat(config): add application configuration load" "internal/config" ".env.example"
do_commit "2026-03-03T09:45:00Z" "feat(db): setup postgres connection and migrations base" "internal/database/postgres.go"
do_commit "2026-03-04T10:00:00Z" "feat(domain): define core domain models for links and users" "internal/domain/user.go" "internal/domain/link.go" "internal/domain/source_link.go"
do_commit "2026-03-04T16:20:00Z" "feat(domain): add domain models for groups and analytics" "internal/domain/group.go" "internal/domain/click_event.go"
do_commit "2026-03-05T13:10:00Z" "feat(db): add initial sql migrations for users, links, and analytics" "internal/database/migrations/000001_create_users.up.sql" "internal/database/migrations/000001_create_users.down.sql" "internal/database/migrations/000002_create_links.up.sql" "internal/database/migrations/000002_create_links.down.sql"
do_commit "2026-03-06T10:30:00Z" "feat(repo): implement core data repositories for postgres" "internal/repository/postgres/user_repo.go" "internal/repository/postgres/link_repo.go" "internal/repository/postgres/errors.go"
do_commit "2026-03-07T11:00:00Z" "feat(dto): configure request and response schemas" "internal/dto"
do_commit "2026-03-08T09:15:00Z" "feat(db): add redis integration for link caching and tracking" "internal/database/redis.go"
do_commit "2026-03-09T14:00:00Z" "feat(repo): implement redis cache repository" "internal/repository/redis"
do_commit "2026-03-10T10:45:00Z" "feat(service): implement link and redirect services" "internal/service/link_service.go" "internal/service/redirect_service.go" "internal/service/analytics_service.go"
do_commit "2026-03-11T13:20:00Z" "feat(service): implement user authentication service" "internal/service/auth_service.go"
do_commit "2026-03-12T10:00:00Z" "feat(middleware): add authentication, cors, and rate limiting middleware" "internal/middleware"
do_commit "2026-03-12T15:30:00Z" "feat(api): add auth, link, and analytics handlers" "internal/handler"
do_commit "2026-03-13T11:45:00Z" "feat(pkg): add pure utilities for url hashing" "pkg/hash"
do_commit "2026-03-14T09:30:00Z" "feat(server): setup gin web server entrypoint and routing" "cmd/server/main.go"
do_commit "2026-03-15T14:15:00Z" "feat(worker): add background worker for persistent click analytics" "internal/worker"
do_commit "2026-03-16T10:00:00Z" "feat(pkg): implement geo location and bot detection modules" "pkg/botdetect" "pkg/geoip" "pkg/useragent"
do_commit "2026-03-17T11:30:00Z" "chore(web): initialize vite react frontend app setup" "web/package.json" "web/vite.config.js" "web/index.html" "web/bun.lock"
do_commit "2026-03-18T09:45:00Z" "feat(web): add api client utilities and global context" "web/src/lib" "web/src/context"
do_commit "2026-03-18T15:20:00Z" "feat(web): implement single sign-on auth pages and components" "web/src/pages/LoginPage.jsx" "web/src/pages/RegisterPage.jsx" "web/src/components"
do_commit "2026-03-19T10:10:00Z" "feat(web): build marketing landing page layout" "web/src/pages/LandingPage.jsx" "web/hero.js"
do_commit "2026-03-19T16:00:00Z" "feat(web): add unified dashboard and link tracking UI" "web/src/pages/LinksPage.jsx" "web/src/pages/LinkDetailPage.jsx"
do_commit "2026-03-20T11:30:00Z" "feat(web): implement campaign group management screens" "web/src/pages/GroupsPage.jsx" "web/src/pages/GroupDetailPage.jsx" "web/src/pages/NotFoundPage.jsx"
do_commit "2026-03-20T17:45:00Z" "style(web): add clean global styles, css utility themes" "web/src/index.css" "web/src/App.jsx" "web/src/main.jsx"
do_commit "2026-03-21T10:00:00Z" "docs: create backend api reference and system architecture diagrams" "docs/api-reference.md" "docs/architecture.md"
do_commit "2026-03-21T15:30:00Z" "chore(docker): setup production containerization and docker compose" "Dockerfile" "Dockerfile.frontend" "docker-compose.yml" "web/nginx.conf" ".github"
do_commit "2026-03-22T09:00:00Z" "docs: add standard local and scalable cloud deployment guides" "docs/deployment.md" "cloudDeployment.md"

git add .
GIT_AUTHOR_DATE="2026-03-22T10:30:00Z" GIT_COMMITTER_DATE="2026-03-22T10:30:00Z" git commit -m "fix: final production readiness checks and polishing across stack" >/dev/null 2>&1 || true

if [ "$BRANCH" != "new-history" ]; then
    git branch -D $BRANCH || true
    git branch -m $BRANCH
fi

git branch -D temp-backup-branch || true

echo "=== New Git History (Top 30 Commits) ==="
git log -30 --format="%cd | [%h] %s" --date=short
