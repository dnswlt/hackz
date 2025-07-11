package rpz

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Item struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
}

type Server struct{}

func (s *Server) Serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /rpz/items/{itemID}", s.handleGetItem)

	log.Println("Listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *Server) handleGetItem(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("itemID")

	item := Item{
		ID:        itemID,
		Name:      "SillyName",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}
