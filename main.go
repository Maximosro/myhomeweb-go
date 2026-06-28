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
	IsBuiltin    bool   `json:"builtin"`
	Links        []Link `json:"links"`
}

type Link struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	Domain       string `json:"domain"`
	CategoryID   string `json:"categoryId"`
	DisplayOrder int    `json:"displayOrder"`
	IsBuiltin    bool   `json:"builtin"`
	CategoryName string `json:"-"` // ponytail: internal, for export
}

type CategoryDTO struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Icon         string    `json:"icon"`
	DisplayOrder int       `json:"displayOrder"`
	Builtin      bool      `json:"builtin"`
	Links        []LinkDTO `json:"links"`
}

type LinkDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	Domain       string `json:"domain"`
	CategoryID   string `json:"categoryId"`
	DisplayOrder int    `json:"displayOrder"`
	Builtin      bool   `json:"builtin"`
}

type ExportDTO struct {
	Categories []CategoryExport `json:"categories"`
	Links      []LinkExport     `json:"links"`
}

type CategoryExport struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type LinkExport struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Category string `json:"category"`
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
	CREATE TABLE IF NOT EXISTS categories (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		icon TEXT NOT NULL,
		display_order INTEGER NOT NULL,
		is_builtin INTEGER NOT NULL
	);
	CREATE TABLE IF NOT EXISTS links (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		category_id TEXT NOT NULL REFERENCES categories(id),
		display_order INTEGER NOT NULL,
		is_builtin INTEGER NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
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

func loadCategories(db *sql.DB) ([]Category, error) {
	rows, err := db.Query(`SELECT id, name, icon, display_order, is_builtin FROM categories ORDER BY display_order ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []Category
	for rows.Next() {
		var c Category
		var builtin int
		if err := rows.Scan(&c.ID, &c.Name, &c.Icon, &c.DisplayOrder, &builtin); err != nil {
			return nil, err
		}
		c.IsBuiltin = builtin == 1
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
		`SELECT id, name, url, category_id, display_order, is_builtin FROM links WHERE category_id = ? ORDER BY display_order ASC`,
		categoryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var l Link
		var builtin int
		if err := rows.Scan(&l.ID, &l.Name, &l.URL, &l.CategoryID, &l.DisplayOrder, &builtin); err != nil {
			return nil, err
		}
		l.Domain = extractDomain(l.URL)
		l.IsBuiltin = builtin == 1
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
		Builtin:      c.IsBuiltin,
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
		Builtin:      l.IsBuiltin,
	}
}

func countCategories(q querier) (int, error) {
	var n int
	err := q.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&n)
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

func getCategoryBuiltin(db *sql.DB, id string) (bool, bool, error) {
	var builtin int
	err := db.QueryRow(`SELECT is_builtin FROM categories WHERE id = ?`, id).Scan(&builtin)
	if err == sql.ErrNoRows {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return builtin == 1, true, nil
}

func getLinkBuiltin(db *sql.DB, id string) (bool, bool, error) {
	var builtin int
	err := db.QueryRow(`SELECT is_builtin FROM links WHERE id = ?`, id).Scan(&builtin)
	if err == sql.ErrNoRows {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return builtin == 1, true, nil
}

func categoryExistsByName(q querier, name string) (bool, error) {
	var n int
	err := q.QueryRow(`SELECT COUNT(*) FROM categories WHERE LOWER(name) = LOWER(?)`, name).Scan(&n)
	return n > 0, err
}

func findCategoryByName(q querier, name string) (*Category, error) {
	var c Category
	var builtin int
	err := q.QueryRow(
		`SELECT id, name, icon, display_order, is_builtin FROM categories WHERE LOWER(name) = LOWER(?) ORDER BY display_order LIMIT 1`,
		name,
	).Scan(&c.ID, &c.Name, &c.Icon, &c.DisplayOrder, &builtin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.IsBuiltin = builtin == 1
	return &c, nil
}

func linkExists(q querier, linkURL, categoryID string) (bool, error) {
	var n int
	err := q.QueryRow(`SELECT COUNT(*) FROM links WHERE url = ? AND category_id = ?`, linkURL, categoryID).Scan(&n)
	return n > 0, err
}

func loadCustomCategories(db *sql.DB) ([]Category, error) {
	rows, err := db.Query(`SELECT id, name, icon, display_order, is_builtin FROM categories WHERE is_builtin = 0 ORDER BY display_order ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cats []Category
	for rows.Next() {
		var c Category
		var builtin int
		if err := rows.Scan(&c.ID, &c.Name, &c.Icon, &c.DisplayOrder, &builtin); err != nil {
			return nil, err
		}
		c.IsBuiltin = builtin == 1
		cats = append(cats, c)
	}
	return cats, nil
}

func loadCustomLinks(db *sql.DB) ([]Link, error) {
	rows, err := db.Query(
		`SELECT l.id, l.name, l.url, l.category_id, l.display_order, l.is_builtin, c.name
		 FROM links l JOIN categories c ON l.category_id = c.id
		 WHERE l.is_builtin = 0 ORDER BY l.display_order ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var links []Link
	for rows.Next() {
		var l Link
		var builtin int
		var catName string
		if err := rows.Scan(&l.ID, &l.Name, &l.URL, &l.CategoryID, &l.DisplayOrder, &builtin, &catName); err != nil {
			return nil, err
		}
		l.Domain = extractDomain(l.URL)
		l.IsBuiltin = builtin == 1
		l.CategoryName = catName
		links = append(links, l)
	}
	return links, nil
}

// ─────────────────── templateData ───────────────────

type dashboardData struct {
	Categories []CategoryDTO
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
		http.ServeFile(w, r, filepath.Join(staticDir, "login.html"))
	})

	// Home (SSR) — serves dashboard if authenticated, login page otherwise
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// ponytail: only handle exact "/", let 404 handler catch the rest
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// Check auth — serve login if no valid session
		token := extractToken(r)
		if token == "" {
			http.ServeFile(w, r, filepath.Join(staticDir, "login.html"))
			return
		}
		if _, err := validateJWT(token); err != nil {
			log.Printf("[auth] JWT validation failed: %v", err)
			http.ServeFile(w, r, filepath.Join(staticDir, "login.html"))
			return
		}

		// Authenticated — render dashboard
		cats, err := loadCategories(db)
		if err != nil {
			log.Printf("[home] error loading categories: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		dtos := make([]CategoryDTO, len(cats))
		for i, c := range cats {
			dtos[i] = categoryToDTO(c)
		}
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "dashboard.html", dashboardData{Categories: dtos}); err != nil {
			log.Printf("[home] template error: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
		buf.WriteTo(w)
	})

	// GET /api/v1/categories
	mux.HandleFunc("GET /api/v1/categories", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		cats, err := loadCategories(db)
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
			Name string `json:"name"`
			Icon string `json:"icon"`
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
		icon := strings.TrimSpace(input.Icon)
		if icon == "" {
			icon = "\U0001F4C1" // 📁
		}
		if len(icon) > maxIconLen {
			jsonError(w, "Icon too long", 400)
			return
		}
		n, err := countCategories(db)
		if err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		displayOrder := n + 1
		id := uuid.New().String()
		_, err = db.Exec(
			`INSERT INTO categories (id, name, icon, display_order, is_builtin) VALUES (?, ?, ?, ?, 0)`,
			id, strings.TrimSpace(input.Name), icon, displayOrder,
		)
		if err != nil {
			jsonError(w, "Failed to create category", 500)
			return
		}
		// Return created DTO — ponytail: build from input+id, skip readback
		dto := CategoryDTO{
			ID:           id,
			Name:         strings.TrimSpace(input.Name),
			Icon:         icon,
			DisplayOrder: displayOrder,
			Builtin:      false,
			Links:        []LinkDTO{},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(dto)
	}))

	// DELETE /api/v1/categories/{id}
	mux.HandleFunc("DELETE /api/v1/categories/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		isBuiltin, found, err := getCategoryBuiltin(db, id)
		if err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		if !found {
			jsonError(w, "Not found", 404)
			return
		}
		if isBuiltin {
			jsonError(w, "Cannot delete built-in category", 403)
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
			`INSERT INTO links (id, name, url, category_id, display_order, is_builtin) VALUES (?, ?, ?, ?, ?, 0)`,
			id, strings.TrimSpace(input.Name), strings.TrimSpace(input.URL), input.CategoryID, maxOrder+1,
		)
		if err != nil {
			jsonError(w, "Failed to create link", 500)
			return
		}
		// Build DTO from input+id, skip readback
		dto := LinkDTO{
			ID:           id,
			Name:         strings.TrimSpace(input.Name),
			URL:          strings.TrimSpace(input.URL),
			Domain:       extractDomain(strings.TrimSpace(input.URL)),
			CategoryID:   input.CategoryID,
			DisplayOrder: maxOrder + 1,
			Builtin:      false,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(dto)
	}))

	// DELETE /api/v1/links/{id}
	mux.HandleFunc("DELETE /api/v1/links/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		isBuiltin, found, err := getLinkBuiltin(db, id)
		if err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		if !found {
			jsonError(w, "Not found", 404)
			return
		}
		if isBuiltin {
			jsonError(w, "Cannot delete built-in link", 403)
			return
		}
		if _, err := db.Exec(`DELETE FROM links WHERE id = ?`, id); err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		w.WriteHeader(204)
	}))

	// GET /api/v1/export
	mux.HandleFunc("GET /api/v1/export", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		cats, err := loadCustomCategories(db)
		if err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		links, err := loadCustomLinks(db)
		if err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		dto := ExportDTO{
			Categories: make([]CategoryExport, len(cats)),
			Links:      make([]LinkExport, len(links)),
		}
		for i, c := range cats {
			dto.Categories[i] = CategoryExport{Name: c.Name, Icon: c.Icon}
		}
		for i, l := range links {
			dto.Links[i] = LinkExport{Name: l.Name, URL: l.URL, Category: l.CategoryName}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dto)
	}))

	// POST /api/v1/import
	mux.HandleFunc("POST /api/v1/import", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		var importData ExportDTO
		if err := json.NewDecoder(r.Body).Decode(&importData); err != nil {
			jsonError(w, "Invalid JSON", 400)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			jsonError(w, "Internal error", 500)
			return
		}
		defer tx.Rollback()

		catsImported := 0
		linksImported := 0
		catMap := make(map[string]string)

		for _, ce := range importData.Categories {
			// Validate lengths
			if len(ce.Name) > maxNameLen || len(ce.Icon) > maxIconLen {
				continue
			}
			exists, err := categoryExistsByName(tx, ce.Name)
			if err != nil {
				jsonError(w, "Internal error", 500)
				return
			}
			if exists {
				cat, err := findCategoryByName(tx, ce.Name)
				if err != nil {
					jsonError(w, "Internal error", 500)
					return
				}
				if cat != nil {
					catMap[strings.ToLower(ce.Name)] = cat.ID
				}
				continue
			}
			n, err := countCategories(tx)
			if err != nil {
				jsonError(w, "Internal error", 500)
				return
			}
			id := uuid.New().String()
			icon := ce.Icon
			if icon == "" {
				icon = "\U0001F4C1"
			}
			if _, err := tx.Exec(`INSERT INTO categories (id, name, icon, display_order, is_builtin) VALUES (?, ?, ?, ?, 0)`,
				id, ce.Name, icon, n+1); err != nil {
				jsonError(w, "Internal error", 500)
				return
			}
			catMap[strings.ToLower(ce.Name)] = id
			catsImported++
		}

		for _, le := range importData.Links {
			// Validate lengths
			if len(le.Name) > maxNameLen || len(le.URL) > maxURLLen || len(le.Category) > maxNameLen {
				continue
			}
			if parsed, err := url.Parse(le.URL); err != nil || parsed.Scheme == "" || parsed.Host == "" {
				continue
			}
			catID, ok := catMap[strings.ToLower(le.Category)]
			if !ok {
				cat, err := findCategoryByName(tx, le.Category)
				if err != nil {
					jsonError(w, "Internal error", 500)
					return
				}
				if cat == nil {
					n, err := countCategories(tx)
					if err != nil {
						jsonError(w, "Internal error", 500)
						return
					}
					id := uuid.New().String()
					if _, err := tx.Exec(`INSERT INTO categories (id, name, icon, display_order, is_builtin) VALUES (?, ?, ?, ?, 0)`,
						id, le.Category, "\U0001F4C1", n+1); err != nil {
						jsonError(w, "Internal error", 500)
						return
					}
					catID = id
					catMap[strings.ToLower(le.Category)] = id
					catsImported++
				} else {
					catID = cat.ID
					catMap[strings.ToLower(le.Category)] = cat.ID
				}
			}
			dup, err := linkExists(tx, le.URL, catID)
			if err != nil {
				jsonError(w, "Internal error", 500)
				return
			}
			if !dup {
				maxOrd, err := maxLinkOrder(tx, catID)
				if err != nil {
					jsonError(w, "Internal error", 500)
					return
				}
				id := uuid.New().String()
				if _, err := tx.Exec(`INSERT INTO links (id, name, url, category_id, display_order, is_builtin) VALUES (?, ?, ?, ?, ?, 0)`,
					id, le.Name, le.URL, catID, maxOrd+1); err != nil {
					jsonError(w, "Internal error", 500)
					return
				}
				linksImported++
			}
		}

		if err := tx.Commit(); err != nil {
			jsonError(w, "Internal error", 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{
			"categoriesImported": catsImported,
			"linksImported":      linksImported,
		})
	}))

	// ─── Start ───
	log.Printf("[server] listening on :%s", port)
	log.Printf("[server] http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
