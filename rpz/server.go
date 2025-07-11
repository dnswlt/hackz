package rpz

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type Item struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
}

type Server struct {
	mu    sync.Mutex
	items map[string]Item
}

func (s *Server) Serve() {
	s.items = make(map[string]Item)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /rpz/items/{itemID}", s.handleGetItem)
	mux.HandleFunc("POST /rpz/items", s.handlePostItem)

	log.Println("Listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *Server) handleGetItem(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("itemID")

	s.mu.Lock()
	item, ok := s.items[itemID]
	s.mu.Unlock()

	if !ok {
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (s *Server) handlePostItem(w http.ResponseWriter, r *http.Request) {
	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	item.Timestamp = time.Now()

	s.mu.Lock()
	s.items[item.ID] = item
	s.mu.Unlock()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}
