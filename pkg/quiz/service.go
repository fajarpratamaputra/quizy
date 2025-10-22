package quiz

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

// Service bundles dependencies required to serve quiz data.
type Service struct {
	db *sql.DB
}

// NewService opens a PostgreSQL connection and performs a health check.
func NewService(dsn string) (*Service, error) {
	if dsn == "" {
		return nil, errors.New("empty DSN")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Connection pool defaults suitable for small workloads.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(10 * time.Second) // wajib pendek

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &Service{db: db}, nil
}

// Close releases database resources.
func (s *Service) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// ServeHTTP satisfies http.Handler and returns paginated quiz data.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	limit, page, err := parsePagination(r)
	if err != nil {
		http.Error(w, `{"error":"invalid pagination"}`, http.StatusBadRequest)
		return
	}
	offset := (page - 1) * limit

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM questions`).Scan(&total); err != nil {
		http.Error(w, `{"error":"failed to count rows"}`, http.StatusInternalServerError)
		log.Printf("count rows: %v", err)
		return
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT question, answer_a, answer_b, answer_c, correct
		FROM questions
		ORDER BY id ASC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		http.Error(w, `{"error":"failed to query data"}`, http.StatusInternalServerError)
		log.Printf("query data: %v", err)
		return
	}
	defer rows.Close()

	items := make([]QuizItem, 0, limit)
	for rows.Next() {
		var q string
		var a, b, c, correct sql.NullString
		if err := rows.Scan(&q, &a, &b, &c, &correct); err != nil {
			http.Error(w, `{"error":"failed to scan row"}`, http.StatusInternalServerError)
			log.Printf("scan row: %v", err)
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
		log.Printf("rows iterate: %v", err)
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
		log.Printf("encode response: %v", err)
		return
	}

	log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
}

// QuizItem mirrors the response payload.
type QuizItem struct {
	Question string   `json:"question"`
	Answers  []string `json:"answers"`
	Correct  string   `json:"correct"`
}

// Meta carries pagination metadata.
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

// QuizResponse combines data and metadata.
type QuizResponse struct {
	Data []QuizItem `json:"data"`
	Meta Meta       `json:"meta"`
}

func parsePagination(r *http.Request) (limit, page int, err error) {
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

// DSNFromEnv fetches the database DSN or returns an error.
func DSNFromEnv() (string, error) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return "", errors.New("missing DB_DSN environment variable")
	}
	return dsn, nil
}
