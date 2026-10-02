package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Suiren91/2vs2shogi_go/internal/ws"
)

type healthResponse struct {
	Status string `json:"status"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(healthResponse{"ok"})
	if err != nil {
		log.Printf("failed to encode respponse: %v", err)
		//HACK: 本当はapplication/jsonで返したい
		http.Error(w, "error", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /ws", ws.Handler)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
