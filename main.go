package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	_ "modernc.org/sqlite"
)

// ─────────────────── Structs ───────────────────

type Category struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	DisplayOrder int    `json:"displayOrder"`
	DashboardID  string `json:"dashboardId"`
	Links        []Link `json:"links"`
}

type Link struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	Domain       string `json:"domain"`
	CategoryID   string `json:"categoryId"`
	DisplayOrder int    `json:"displayOrder"`
}

type Dashboard struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DisplayOrder int    `json:"displayOrder"`
}

type DashboardDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DisplayOrder int    `json:"displayOrder"`
}

type CategoryDTO struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Icon         string    `json:"icon"`
	DisplayOrder int       `json:"displayOrder"`
	Links        []LinkDTO `json:"links"`
}

type LinkDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	Domain       string `json:"domain"`
	CategoryID   string `json:"categoryId"`
	DisplayOrder int    `json:"displayOrder"`
}

// ─────────────────── Helpers ───────────────────

const (
	maxNameLen = 200
	maxURLLen  = 2048
	maxIconLen = 50

	supabaseURL   = "https://agtkcnxmlbccbwmsuxdz.supabase.co"
	jwksURL       = supabaseURL + "/auth/v1/.well-known/jwks.json"
	jwksCacheTime = 1 * time.Hour
)

func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Host
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ─────────────────── DB ───────────────────

func initDB(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("pragma %s: %w", p, err)
		}
	}

	schema := `
	CREATE TABLE IF NOT EXISTS dashboards (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		display_order INTEGER NOT NULL
	);
	CREATE TABLE IF NOT EXISTS categories (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		icon TEXT NOT NULL,
		display_order INTEGER NOT NULL,
		is_builtin INTEGER NOT NULL DEFAULT 0,
		dashboard_id TEXT NOT NULL DEFAULT 'd0000000-0000-0000-0000-000000000001' REFERENCES dashboards(id)
	);
	CREATE TABLE IF NOT EXISTS links (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		category_id TEXT NOT NULL REFERENCES categories(id),
		display_order INTEGER NOT NULL,
		is_builtin INTEGER NOT NULL DEFAULT 0
	);
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	// Migration: add dashboard_id to existing categories (idempotent)
	// ponytail: no REFERENCES in ALTER TABLE — SQLite rejects it with non-NULL default
	if _, err := db.Exec(`ALTER TABLE categories ADD COLUMN dashboard_id TEXT NOT NULL DEFAULT 'd0000000-0000-0000-0000-000000000001'`); err != nil {
		// ponytail: "duplicate column name" means already migrated — that's OK
		if !strings.Contains(err.Error(), "duplicate column name") {
			db.Close()
			return nil, fmt.Errorf("migration add dashboard_id: %w", err)
		}
	}

	return db, nil
}

func seedDB(db *sql.DB, seedPath string) {
	data, err := os.ReadFile(seedPath)
	if err != nil {
		log.Printf("[seed] data.sql not found at %s — skipping seed (DB may be empty)", seedPath)
		return
	}
	if _, err := db.Exec(string(data)); err != nil {
		log.Printf("[seed] error executing data.sql: %v", err)
		return
	}
	log.Println("[seed] data.sql executed successfully")
}

// ─────────────────── DB Queries ───────────────────

// querier is satisfied by both *sql.DB and *sql.Tx
type querier interface {
	QueryRow(query string, args ...any) *sql.Row
}

func loadCategories(db *sql.DB, dashboardID string) ([]Category, error) {
	rows, err := db.Query(
		`SELECT id, name, icon, display_order, dashboard_id FROM categories WHERE dashboard_id = ? ORDER BY display_order ASC`,
		dashboardID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Icon, &c.DisplayOrder, &c.DashboardID); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}

	for i := range cats {
		links, err := loadLinksByCategory(db, cats[i].ID)
		if err != nil {
			return nil, err
		}
		cats[i].Links = links
	}
	return cats, nil
}

func loadLinksByCategory(db *sql.DB, categoryID string) ([]Link, error) {
	rows, err := db.Query(
		`SELECT id, name, url, category_id, display_order FROM links WHERE category_id = ? ORDER BY display_order ASC`,
		categoryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.ID, &l.Name, &l.URL, &l.CategoryID, &l.DisplayOrder); err != nil {
			return nil, err
		}
		l.Domain = extractDomain(l.URL)
		links = append(links, l)
	}
	if links == nil {
		links = []Link{} // ponytail: json [] not null
	}
	return links, nil
}

func categoryToDTO(c Category) CategoryDTO {
	dto := CategoryDTO{
		ID:           c.ID,
		Name:         c.Name,
		Icon:         c.Icon,
		DisplayOrder: c.DisplayOrder,
		Links:        make([]LinkDTO, len(c.Links)),
	}
	for i, l := range c.Links {
		dto.Links[i] = linkToDTO(l)
	}
	if dto.Links == nil {
		dto.Links = []LinkDTO{}
	}
	return dto
}

func linkToDTO(l Link) LinkDTO {
	return LinkDTO{
		ID:           l.ID,
		Name:         l.Name,
		URL:          l.URL,
		Domain:       l.Domain,
		CategoryID:   l.CategoryID,
		DisplayOrder: l.DisplayOrder,
	}
}

func countCategories(q querier, dashboardID string) (int, error) {
	var n int
	err := q.QueryRow(`SELECT COUNT(*) FROM categories WHERE dashboard_id = ?`, dashboardID).Scan(&n)
	return n, err
}

func maxLinkOrder(q querier, categoryID string) (int, error) {
	var n sql.NullInt64
	err := q.QueryRow(`SELECT MAX(display_order) FROM links WHERE category_id = ?`, categoryID).Scan(&n)
	if err != nil {
		return 0, err
	}
	if n.Valid {
		return int(n.Int64), nil
	}
	return 0, nil
}

func categoryExists(db *sql.DB, id string) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM categories WHERE id = ?`, id).Scan(&n)
	return n > 0, err
}

func dashboardExists(db *sql.DB, id string) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM dashboards WHERE id = ?`, id).Scan(&n)
	return n > 0, err
}

func loadDashboards(db *sql.DB) ([]Dashboard, error) {
	rows, err := db.Query(`SELECT id, name, display_order FROM dashboards ORDER BY display_order ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dashboards []Dashboard
	for rows.Next() {
		var d Dashboard
		if err := rows.Scan(&d.ID, &d.Name, &d.DisplayOrder); err != nil {
			return nil, err
		}
		dashboards = append(dashboards, d)
	}
	return dashboards, nil
}

func countDashboards(q querier) (int, error) {
	var n int
	err := q.QueryRow(`SELECT COUNT(*) FROM dashboards`).Scan(&n)
	return n, err
}

// ─────────────────── templateData ───────────────────

type dashboardData struct {
	Dashboards        []DashboardDTO
	ActiveDashboardID string
	Categories        []CategoryDTO
}

// ─────────────────── Auth ───────────────────

type contextKey string

const userIDKey contextKey = "user_id"

var (
	jwksCache   jwk.Set
	jwksCacheMu sync.RWMutex
	jwksFetched time.Time
)

func getJWKS() (jwk.Set, error) {
	jwksCacheMu.RLock()
	if jwksCache != nil && time.Since(jwksFetched) < jwksCacheTime {
		set := jwksCache
		jwksCacheMu.RUnlock()
		return set, nil
	}
	jwksCacheMu.RUnlock()

	jwksCacheMu.Lock()
	defer jwksCacheMu.Unlock()

	if jwksCache != nil && time.Since(jwksFetched) < jwksCacheTime {
		return jwksCache, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	set, err := jwk.Fetch(ctx, jwksURL)
	if err != nil {
		if jwksCache != nil {
			log.Printf("[auth] JWKS fetch failed, using stale cache: %v", err)
			return jwksCache, nil
		}
		return nil, fmt.Errorf("jwks fetch: %w", err)
	}
	jwksCache = set
	jwksFetched = time.Now()
	return set, nil
}

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if after, found := strings.CutPrefix(auth, "Bearer "); found {
		return after
	}
	cookie, err := r.Cookie("sb-access-token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

func validateJWT(tokenStr string) (string, error) {
	set, err := getJWKS()
	if err != nil {
		log.Printf("[auth] JWKS fetch failed: %v", err)
		return "", err
	}

	log.Printf("[auth] JWKS retrieved, keys: %d", set.Len())
	parsed, err := jwt.Parse([]byte(tokenStr),
		jwt.WithKeySet(set),
		jwt.WithAcceptableSkew(1*time.Minute),
		jwt.WithValidate(true),
	)
	if err != nil {
		log.Printf("[auth] JWT parse/validate failed: %v (token prefix: %.20s...)", err, tokenStr)
		return "", err
	}

	var sub string
	if err := parsed.Get("sub", &sub); err != nil {
		sub = "unknown"
	}
	return sub, nil
}

// serveLogin sets no-cache and serves login.html to prevent browser caching stale code
func serveLogin(w http.ResponseWriter, r *http.Request, staticDir string) {
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, filepath.Join(staticDir, "login.html"))
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		userID, err := validateJWT(token)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or expired token"})
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

// ─────────────────── Main ───────────────────

func main() {
	port := getEnv("PORT", "19484")
	dbPath := getEnv("DB_PATH", "data/myhomeweb.db")
	staticDir := getEnv("STATIC_DIR", "static")
	templateDir := getEnv("TEMPLATE_DIR", "templates")
	seedSQL := getEnv("SEED_SQL", "data.sql")

	// Init DB
	db, err := initDB(dbPath)
	if err != nil {
		log.Fatalf("DB init failed: %v", err)
	}
	defer db.Close()
	log.Printf("[db] connected to %s", dbPath)

	// Seed
	seedDB(db, seedSQL)

	// Parse templates
	tmpl, err := template.ParseGlob(filepath.Join(templateDir, "*.html"))
	if err != nil {
		log.Fatalf("Template parse failed: %v", err)
	}

	// ─── Routes ───
	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Health check (public, no auth)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Login page (public)
	mux.HandleFunc("GET /login.html", func(w http.ResponseWriter, r *http.Request) {
		serveLogin(w, r, staticDir)
	})

	// Home (SSR) — redirect to first dashboard if authenticated, login page otherwise
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// ponytail: only handle exact "/", let 404 handler catch the rest
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// ponytail: accept token via query param (post-login redirect), set HTTP cookie, redirect clean
		if tokenParam := r.URL.Query().Get("token"); tokenParam != "" {
			if _, err := validateJWT(tokenParam); err != nil {
				log.Printf("[auth] token from query param invalid: %v", err)
				serveLogin(w, r, staticDir)
				return
			}
			log.Printf("[auth] token from query param valid, setting cookie and redirecting to /")
			http.SetCookie(w, &http.Cookie{
				Name:     "sb-access-token",
				Value:    tokenParam,
				Path:     "/",
				MaxAge:   3600,
				SameSite: http.SameSiteLaxMode,
			})
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// Check auth — serve login if no valid session
		token := extractToken(r)
		if token == "" {
			serveLogin(w, r, staticDir)
			return
		}
		if _, err := validateJWT(token); err != nil {
			serveLogin(w, r, staticDir)
			return
		}

		// Authenticated — redirect to first dashboard
		dashboards, err := loadDashboards(db)
		if err != nil || len(dashboards) == 0 {
			http.Redirect(w, r, "/dashboard/_empty", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/dashboard/"+dashboards[0].ID, http.StatusFound)
	})

	// Dashboard SSR
	mux.HandleFunc("GET /dashboard/{id}", func(w http.ResponseWriter, r *http.Request) {
		dashboardID := r.PathValue("id")

		// Check auth
		token := extractToken(r)
		if token == "" {
			serveLogin(w, r, staticDir)
			return
		}
		if _, err := validateJWT(token); err != nil {
			serveLogin(w, r, staticDir)
			return
		}

		// Load dashboards for tabs
		dashboards, err := loadDashboards(db)
		if err != nil {
			log.Printf("[dashboard] error loading dashboards: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		// Load categories for this dashboard
		var cats []Category
		if dashboardID != "_empty" {
			cats, err = loadCategories(db, dashboardID)
			if err != nil {
				log.Printf("[dashboard] error loading categories: %v", err)
				http.Error(w, "Internal Server Error", 500)
				return
			}
		}

		// Build DTOs
		dashDTOs := make([]DashboardDTO, len(dashboards))
		for i, d := range dashboards {
			dashDTOs[i] = DashboardDTO{ID: d.ID, Name: d.Name, DisplayOrder: d.DisplayOrder}
		}
		catDTOs := make([]CategoryDTO, len(cats))
		for i, c := range cats {
			catDTOs[i] = categoryToDTO(c)
		}

		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "dashboard.html", dashboardData{
			Dashboards:        dashDTOs,
			ActiveDashboardID: dashboardID,
			Categories:        catDTOs,
		}); err != nil {
			log.Printf("[dashboard] template error: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		buf.WriteTo(w)
	})

	// ─── Dashboard API ───

	// GET /api/v1/dashboards
	mux.HandleFunc("GET /api/v1/dashboards", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		dashboards, err := loadDashboards(db)
		if err != nil {
			jsonError(w, "Internal server error", 500)
			return
		}
		dtos := make([]DashboardDTO, len(dashboards))
		for i, d := range dashboards {
			dtos[i] = DashboardDTO{ID: d.ID, Name: d.Name, DisplayOrder: d.DisplayOrder}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dtos)
	}))

	// POST /api/v1/dashboards
	mux.HandleFunc("POST /api/v1/dashboards", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		var input struct{ Name string `json:"name"` }
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			jsonError(w, "Invalid JSON", 400)
			return
		}
		if strings.TrimSpace(input.Name) == "" {
			jsonError(w, "Name is required", 400)
			return
		}
		if len(input.Name) > maxNameLen {
			jsonError(w, "Name too long", 400)
			return
		}
		n, err := countDashboards(db)
		if err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		id := uuid.New().String()
		_, err = db.Exec(`INSERT INTO dashboards (id, name, display_order) VALUES (?, ?, ?)`,
			id, strings.TrimSpace(input.Name), n+1)
		if err != nil {
			jsonError(w, "Failed to create dashboard", 500)
			return
		}
		dto := DashboardDTO{ID: id, Name: strings.TrimSpace(input.Name), DisplayOrder: n + 1}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(dto)
	}))

	// PUT /api/v1/dashboards/{id}
	mux.HandleFunc("PUT /api/v1/dashboards/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		exists, err := dashboardExists(db, id)
		if err != nil || !exists {
			jsonError(w, "Not found", 404)
			return
		}
		var input struct{ Name string `json:"name"` }
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			jsonError(w, "Invalid JSON", 400)
			return
		}
		if strings.TrimSpace(input.Name) == "" {
			jsonError(w, "Name is required", 400)
			return
		}
		if len(input.Name) > maxNameLen {
			jsonError(w, "Name too long", 400)
			return
		}
		_, err = db.Exec(`UPDATE dashboards SET name = ? WHERE id = ?`, strings.TrimSpace(input.Name), id)
		if err != nil {
			jsonError(w, "Failed to update dashboard", 500)
			return
		}
		dto := DashboardDTO{ID: id, Name: strings.TrimSpace(input.Name)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dto)
	}))

	// DELETE /api/v1/dashboards/{id}
	mux.HandleFunc("DELETE /api/v1/dashboards/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		exists, err := dashboardExists(db, id)
		if err != nil || !exists {
			jsonError(w, "Not found", 404)
			return
		}
		tx, err := db.Begin()
		if err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		defer tx.Rollback()
		if _, err := tx.Exec(`DELETE FROM links WHERE category_id IN (SELECT id FROM categories WHERE dashboard_id = ?)`, id); err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		if _, err := tx.Exec(`DELETE FROM categories WHERE dashboard_id = ?`, id); err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		if _, err := tx.Exec(`DELETE FROM dashboards WHERE id = ?`, id); err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		if err := tx.Commit(); err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		w.WriteHeader(204)
	}))

	// ─── Category API ───

	// GET /api/v1/categories
	mux.HandleFunc("GET /api/v1/categories", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		dashboardID := r.URL.Query().Get("dashboard_id")
		if dashboardID == "" {
			jsonError(w, "dashboard_id query param is required", 400)
			return
		}
		cats, err := loadCategories(db, dashboardID)
		if err != nil {
			jsonError(w, "Internal server error", 500)
			return
		}
		dtos := make([]CategoryDTO, len(cats))
		for i, c := range cats {
			dtos[i] = categoryToDTO(c)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dtos)
	}))

	// POST /api/v1/categories
	mux.HandleFunc("POST /api/v1/categories", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name        string `json:"name"`
			Icon        string `json:"icon"`
			DashboardID string `json:"dashboardId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			jsonError(w, "Invalid JSON", 400)
			return
		}
		if strings.TrimSpace(input.Name) == "" {
			jsonError(w, "Name is required", 400)
			return
		}
		if len(input.Name) > maxNameLen {
			jsonError(w, "Name too long", 400)
			return
		}
		if strings.TrimSpace(input.DashboardID) == "" {
			jsonError(w, "dashboardId is required", 400)
			return
		}
		exists, err := dashboardExists(db, input.DashboardID)
		if err != nil || !exists {
			jsonError(w, "Dashboard not found", 400)
			return
		}
		icon := strings.TrimSpace(input.Icon)
		if icon == "" {
			icon = "\U0001F4C1" // 📁
		}
		if len(icon) > maxIconLen {
			jsonError(w, "Icon too long", 400)
			return
		}
		n, err := countCategories(db, input.DashboardID)
		if err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		displayOrder := n + 1
		id := uuid.New().String()
		_, err = db.Exec(
			`INSERT INTO categories (id, name, icon, display_order, dashboard_id) VALUES (?, ?, ?, ?, ?)`,
			id, strings.TrimSpace(input.Name), icon, displayOrder, input.DashboardID,
		)
		if err != nil {
			jsonError(w, "Failed to create category", 500)
			return
		}
		dto := CategoryDTO{
			ID:           id,
			Name:         strings.TrimSpace(input.Name),
			Icon:         icon,
			DisplayOrder: displayOrder,
			Links:        []LinkDTO{},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(dto)
	}))

	// DELETE /api/v1/categories/{id}
	mux.HandleFunc("DELETE /api/v1/categories/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		exists, err := categoryExists(db, id)
		if err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		if !exists {
			jsonError(w, "Not found", 404)
			return
		}
		tx, err := db.Begin()
		if err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		defer tx.Rollback()
		if _, err := tx.Exec(`DELETE FROM links WHERE category_id = ?`, id); err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		if _, err := tx.Exec(`DELETE FROM categories WHERE id = ?`, id); err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		if err := tx.Commit(); err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		w.WriteHeader(204)
	}))

	// ─── Link API ───

	// POST /api/v1/links
	mux.HandleFunc("POST /api/v1/links", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name       string `json:"name"`
			URL        string `json:"url"`
			CategoryID string `json:"categoryId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			jsonError(w, "Invalid JSON", 400)
			return
		}
		if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.URL) == "" || strings.TrimSpace(input.CategoryID) == "" {
			jsonError(w, "name, url and categoryId are required", 400)
			return
		}
		if len(input.Name) > maxNameLen {
			jsonError(w, "Name too long", 400)
			return
		}
		if len(input.URL) > maxURLLen {
			jsonError(w, "URL too long", 400)
			return
		}
		parsedURL, err := url.Parse(input.URL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			jsonError(w, "Invalid URL", 400)
			return
		}
		exists, err := categoryExists(db, input.CategoryID)
		if err != nil || !exists {
			jsonError(w, "Category not found", 400)
			return
		}
		maxOrder, err := maxLinkOrder(db, input.CategoryID)
		if err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		id := uuid.New().String()
		_, err = db.Exec(
			`INSERT INTO links (id, name, url, category_id, display_order) VALUES (?, ?, ?, ?, ?)`,
			id, strings.TrimSpace(input.Name), strings.TrimSpace(input.URL), input.CategoryID, maxOrder+1,
		)
		if err != nil {
			jsonError(w, "Failed to create link", 500)
			return
		}
		dto := LinkDTO{
			ID:           id,
			Name:         strings.TrimSpace(input.Name),
			URL:          strings.TrimSpace(input.URL),
			Domain:       extractDomain(strings.TrimSpace(input.URL)),
			CategoryID:   input.CategoryID,
			DisplayOrder: maxOrder + 1,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(dto)
	}))

	// DELETE /api/v1/links/{id}
	mux.HandleFunc("DELETE /api/v1/links/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var n int
		err := db.QueryRow(`SELECT COUNT(*) FROM links WHERE id = ?`, id).Scan(&n)
		if err != nil || n == 0 {
			jsonError(w, "Not found", 404)
			return
		}
		if _, err := db.Exec(`DELETE FROM links WHERE id = ?`, id); err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		w.WriteHeader(204)
	}))

	log.Printf("[server] listening on :%s", port)
	log.Printf("[server] http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
