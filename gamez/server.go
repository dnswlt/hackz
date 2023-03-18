package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	gameHtmlFile  = flag.String("html", "index.html", "Path to game HTML file")
	serverAddress = flag.String("address", "", "Address on which to listen")
	serverPort    = flag.Int("port", 8084, "Port on which to listen")
)

func readGameHtml() (string, error) {
	html, err := os.ReadFile(*gameHtmlFile)
	if err != nil {
		return "", err
	}
	return string(html), nil
}

func gameHandler(w http.ResponseWriter, r *http.Request) {
	gameHtml, err := readGameHtml()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, gameHtml)
}

type MoveRequest struct {
	Move int `json:"move"`
}
type MoveResponse struct {
	Move int `json:"move"`
}

func moveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusBadRequest)
		return
	}
	dec := json.NewDecoder(r.Body)
	var req MoveRequest
	if err := dec.Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	log.Print("Received move request with value ", req.Move)
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	if err := enc.Encode(MoveResponse{Move: req.Move}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type ServerEvent struct {
	Timestamp string `json:"timestamp"`
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	for {
		tick := time.After(1 * time.Second)
		select {
		case <-tick:
			ev := ServerEvent{Timestamp: time.Now().Format(time.RFC3339)}
			var buf strings.Builder
			enc := json.NewEncoder(&buf)
			if err := enc.Encode(ev); err != nil {
				http.Error(w, "Serialization error", http.StatusInternalServerError)
				panic(fmt.Sprintf("Cannot serialize my own structs?! %s", err))
			}
			fmt.Fprintf(w, "data: %s\n\n", buf.String())
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			log.Printf("Client %s disconnected", r.RemoteAddr)
			return
		}
	}
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("Requested invalid path: ", r.URL.Path, r.URL.RawQuery)
}

func main() {
	flag.Parse()
	// Make sure we have access to the game HTML file.
	if _, err := readGameHtml(); err != nil {
		log.Fatal("Cannot load game HTML: ", err)
	}
	http.HandleFunc("/hexz", gameHandler)
	http.HandleFunc("/hexz/move", moveHandler)
	http.HandleFunc("/hexz/sse", sseHandler)
	http.HandleFunc("/", defaultHandler)

	addr := fmt.Sprintf("%s:%d", *serverAddress, *serverPort)
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
