package handler

import (
	"log"
	"net/http"
	"sync"

	"quizy/pkg/quiz"
)

var (
	svc   *quiz.Service
	svcMu sync.Mutex
)

func getService() (*quiz.Service, error) {
	svcMu.Lock()
	defer svcMu.Unlock()

	if svc != nil {
		return svc, nil
	}

	dsn, err := quiz.DSNFromEnv()
	if err != nil {
		return nil, err
	}

	service, err := quiz.NewService(dsn)
	if err != nil {
		return nil, err
	}

	svc = service
	return svc, nil
}

// Handler is the Vercel entrypoint.
func Handler(w http.ResponseWriter, r *http.Request) {
	service, err := getService()
	if err != nil {
		http.Error(w, `{"error":"service initialization failed"}`, http.StatusInternalServerError)
		log.Printf("handler init: %v", err)
		return
	}

	service.ServeHTTP(w, r)
}
