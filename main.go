package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"quizy/pkg/quiz"
)

func main() {
	_ = godotenv.Load()

	dsn, err := quiz.DSNFromEnv()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	svc, err := quiz.NewService(dsn)
	if err != nil {
		log.Fatalf("service init: %v", err)
	}
	defer svc.Close()

	mux := http.NewServeMux()
	mux.Handle("/api/quiz", svc)

	addr := defaultAddr()
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func defaultAddr() string {
	if v := os.Getenv("PORT"); v != "" {
		return ":" + v
	}
	return ":8080"
}
