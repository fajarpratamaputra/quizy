package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type QuizItem struct {
	Question string   `json:"question"`
	Answers  []string `json:"answers"`
	Correct  string   `json:"correct"`
}

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

type QuizResponse struct {
	Data []QuizItem `json:"data"`
	Meta Meta       `json:"meta"`
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env: %s", key)
	}
	return v
}

func main() {
	_ = godotenv.Load()
	// ENV:
	//   DB_DSN="postgres://user:pass@localhost:5432/yourdb?sslmode=disable"
	dsn := mustGetEnv("DB_DSN")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Connection pool settings (aman untuk prod kecil)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	// Health check on start
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("db ping failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/quiz", func(w http.ResponseWriter, r *http.Request) {
		handleQuiz(w, r, db)
	})

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, withJSON(mux)); err != nil {
		log.Fatal(err)
	}
}

// Middleware sederhana untuk set JSON header & basic logging
func withJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func handleQuiz(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Query params
	limit, page, err := parsePagination(r)
	if err != nil {
		http.Error(w, `{"error":"invalid pagination"}`, http.StatusBadRequest)
		return
	}
	offset := (page - 1) * limit

	// Total rows
	var total int
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM questions`).Scan(&total); err != nil {
		http.Error(w, `{"error":"failed to count rows"}`, http.StatusInternalServerError)
		return
	}

	// Query page
	rows, err := db.QueryContext(ctx, `
		SELECT question, answer_a, answer_b, answer_c, correct
		FROM questions
		ORDER BY id ASC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		http.Error(w, `{"error":"failed to query data"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := make([]QuizItem, 0, limit)
	for rows.Next() {
		var q string
		var a, b, c, correct sql.NullString
		if err := rows.Scan(&q, &a, &b, &c, &correct); err != nil {
			http.Error(w, `{"error":"failed to scan row"}`, http.StatusInternalServerError)
			return
		}
		item := QuizItem{
			Question: q,
			Answers: []string{
				nullToString(a),
				nullToString(b),
				nullToString(c),
			},
			Correct: nullToString(correct),
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, `{"error":"rows error"}`, http.StatusInternalServerError)
		return
	}

	totalPages := calcTotalPages(total, limit)
	resp := QuizResponse{
		Data: items,
		Meta: Meta{
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	enc.SetIndent("", "  ")
	if err := enc.Encode(resp); err != nil {
		http.Error(w, `{"error":"encode error"}`, http.StatusInternalServerError)
		return
	}
}

func parsePagination(r *http.Request) (limit, page int, err error) {
	// Default
	limit = 10
	page = 1

	if v := r.URL.Query().Get("limit"); v != "" {
		limit, err = strconv.Atoi(v)
		if err != nil || limit <= 0 {
			return 0, 0, errors.New("bad limit")
		}
	}
	if v := r.URL.Query().Get("page"); v != "" {
		page, err = strconv.Atoi(v)
		if err != nil || page <= 0 {
			return 0, 0, errors.New("bad page")
		}
	}

	// Guardrail
	if limit > 100 {
		limit = 100
	}

	return limit, page, nil
}

func calcTotalPages(total, limit int) int {
	if total == 0 {
		return 0
	}
	pages := total / limit
	if total%limit != 0 {
		pages++
	}
	return pages
}

func nullToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
