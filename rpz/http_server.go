package rpz

import (
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/dnswlt/hackz/rpz/rpzpb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type HTTPServer struct {
	config Config
	mu     sync.Mutex
	items  map[string]*rpzpb.Item
}

func NewHTTPServer(config Config) *HTTPServer {
	return &HTTPServer{
		config: config,
		items:  make(map[string]*rpzpb.Item),
	}
}

func (s *HTTPServer) Serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /rpz/items/{itemID}", s.handleGetItem)
	mux.HandleFunc("POST /rpz/items", s.handlePostItem)

	if !s.config.Insecure {
		// TLS
		log.Println("Listening on :8443")
		if err := http.ListenAndServeTLS(":8443", s.config.CertFile, s.config.KeyFile, mux); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	} else {
		// No TLS. Insecure!
		log.Println("Listening on :8080")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
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
	out, err := protojson.Marshal(item)
	if err != nil {
		log.Fatalf("Cannot marshal own proto as json: %v", err)
	}
	w.Write(out)
}

func (s *HTTPServer) handlePostItem(w http.ResponseWriter, r *http.Request) {
	var item rpzpb.Item

	defer r.Body.Close()
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read http request body", http.StatusInternalServerError)
		return
	}

	if err := protojson.Unmarshal(bs, &item); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	item.Timestamp = timestamppb.Now()
	if s.config.PayloadBytes > 0 {
		item.Payload = randomString(s.config.PayloadBytes)
	}

	s.mu.Lock()
	s.items[item.Id] = &item
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	out, err := protojson.Marshal(&item)
	if err != nil {
		log.Fatalf("Cannot marshal own proto as json: %v", err)
	}
	w.Write(out)
}
