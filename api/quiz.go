package main

import (
	"log"
	"net/http"
	"sync"

	"quizy/internal/quiz"
)

var (
	once sync.Once
	svc  *quiz.Service
	err  error
)

// Handler is the Vercel entrypoint.
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		var dsn string
		dsn, err = quiz.DSNFromEnv()
		if err != nil {
			return
		}
		svc, err = quiz.NewService(dsn)
	})

	if err != nil {
		http.Error(w, `{"error":"service initialization failed"}`, http.StatusInternalServerError)
		log.Printf("handler init: %v", err)
		return
	}

	svc.ServeHTTP(w, r)
}
