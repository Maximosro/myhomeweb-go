# myhomeweb

Personal dashboard for self-hosted apps and links — Go port from Spring Boot/Java.

**Live:** [home.srv1158554.hstgr.cloud](https://home.srv1158554.hstgr.cloud)

[![Deploy to Hostinger KVM](https://github.com/Maximosro/myhomeweb-go/actions/workflows/docker-build-deploy.yml/badge.svg)](https://github.com/Maximosro/myhomeweb-go/actions/workflows/docker-build-deploy.yml)

---

## Stack

| Layer | Tech |
|---|---|
| Backend | Go 1.26, `net/http`, `html/template` |
| Database | SQLite via `modernc.org/sqlite` (pure Go, no CGO) |
| Auth | Supabase JWT via `go-jwx/v3` |
| Frontend | Vanilla JS + CSS (dark theme, glassmorphism) |
| Deploy | Docker multi-stage → GHCR → Hostinger VPS |

## Quick start

```bash
go build -o myhomeweb .
./myhomeweb
# → http://localhost:19484
```

### Env vars

| Variable | Default |
|---|---|
| `PORT` | `19484` |
| `DB_PATH` | `data/myhomeweb.db` |
| `STATIC_DIR` | `static` |
| `TEMPLATE_DIR` | `templates` |
| `SEED_SQL` | `data.sql` |

### Docker

```bash
docker build -t myhomeweb .
docker run -p 19484:19484 -v ./data:/data myhomeweb
```

## API

All write endpoints require Supabase JWT (`Authorization: Bearer …` or `sb-access-token` cookie).

| Method | Endpoint | Notes |
|---|---|---|
| `GET /` | Dashboard (SSR) | Redirects to login if unauthenticated |
| `GET /health` | Health check | Public |
| `GET /api/v1/categories` | List all | Includes nested links |
| `POST /api/v1/categories` | Create | `{name, icon?}` |
| `DELETE /api/v1/categories/{id}` | Delete | 403 if built-in |
| `POST /api/v1/links` | Create | `{name, url, categoryId}` |
| `DELETE /api/v1/links/{id}` | Delete | 403 if built-in |
| `GET /api/v1/export` | Export custom data | JSON |
| `POST /api/v1/import` | Import | Dedup by name/URL |

## Architecture

Single-file backend (`main.go`). No frameworks, no ORM. Statically compiled binary.

- **9 built-in categories**, **43 built-in links** — seeded from `data.sql`
- Custom categories/links are user-owned, exportable/importable
- Built-in items are protected from deletion
- Auth via Supabase JWKS with 1h cache + stale fallback

## CI/CD

- **PRs** → `go test ./...` + build check
- **Push to `main`** → Docker build → push to `ghcr.io` → SSH deploy to VPS → smoke test

## Git workflow

GitFlow: `main` (prod) → `develop` (integration) → `feature/*`

## Original project

Java/Spring Boot version at [Maximosro/myhomeweb](https://github.com/Maximosro/myhomeweb).
