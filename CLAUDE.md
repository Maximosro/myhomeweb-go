# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Build
go build -o myhomeweb .

# Run (development)
go run .
# → http://localhost:19484

# Run (production)
./myhomeweb
```

## Project Architecture

**Stack:** Go 1.24+ / SQLite (modernc.org/sqlite, pure Go) / html/template / vanilla JS + CSS.

Single-file backend: `main.go` (~400 lines). No frameworks, no ORM.

### Structure

| File/Dir | Purpose |
|---|---|
| `main.go` | Entire backend: structs, DB init, seed, HTTP handlers, templates |
| `go.mod` | Go module + dependencies |
| `data.sql` | Seed data (9 built-in categories, 43 built-in links) |
| `data/` | SQLite DB file (gitignored) |
| `templates/dashboard.html` | SSR template (Go html/template) |
| `static/` | CSS, JS, images (served as-is) |

### API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET /` | SSR dashboard | Renders all categories + links |
| `GET /api/v1/categories` | List categories | Includes nested links |
| `POST /api/v1/categories` | Create category | `{name, icon?}` |
| `DELETE /api/v1/categories/{id}` | Delete category | 403 if builtin |
| `POST /api/v1/links` | Create link | `{name, url, categoryId}` |
| `DELETE /api/v1/links/{id}` | Delete link | 403 if builtin |
| `GET /api/v1/export` | Export custom data | JSON |
| `POST /api/v1/import` | Import custom data | Dedup by name/URL |

### Configuration

Environment variables with defaults:

| Variable | Default |
|---|---|
| `PORT` | `19484` |
| `DB_PATH` | `data/myhomeweb.db` |
| `STATIC_DIR` | `static` |
| `TEMPLATE_DIR` | `templates` |
| `SEED_SQL` | `data.sql` |

### Frontend

- `static/css/dashboard.css` — dark theme, glassmorphism, Inter + Orbitron fonts
- `static/js/dashboard.js` — vanilla JS: weather (Open-Meteo), bandwidth (Cloudflare), app status checker, CRUD modals, JSON export/import

### Git workflow

Follows GitFlow: `main` (production) → `develop` (integration) → `feature/*`.

### Java legacy

Original Spring Boot codebase at [Maximosro/myhomeweb](https://github.com/Maximosro/myhomeweb).
