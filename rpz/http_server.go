package rpz

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type HTTPServer struct {
	mu    sync.Mutex
	items map[string]Item
}

type Item struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
}

func (s *HTTPServer) Serve() {
	s.items = make(map[string]Item)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /rpz/items/{itemID}", s.handleGetItem)
	mux.HandleFunc("POST /rpz/items", s.handlePostItem)

	log.Println("Listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *HTTPServer) ServeTLS(certFile, keyFile string) {
	s.items = make(map[string]Item)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /rpz/items/{itemID}", s.handleGetItem)
	mux.HandleFunc("POST /rpz/items", s.handlePostItem)

	log.Println("Listening on :8443")
	if err := http.ListenAndServeTLS(":8443", certFile, keyFile, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *HTTPServer) handleGetItem(w http.ResponseWriter, r *http.Request) {
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

func (s *HTTPServer) handlePostItem(w http.ResponseWriter, r *http.Request) {
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
