package main

import (
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	gameHtmlFile  = flag.String("html", "index.html", "Path to game HTML file")
	serverAddress = flag.String("address", "", "Address on which to listen")
	serverPort    = flag.Int("port", 8084, "Port on which to listen")

	ongoingGames    = make(map[string]*Game)
	ongoingGamesMut sync.Mutex
)

func readGameHtml() (string, error) {
	html, err := os.ReadFile(*gameHtmlFile)
	if err != nil {
		return "", err
	}
	return string(html), nil
}

type Game struct {
	Id          string
	ControlChan chan ControlEvent // The channel to communicate with the game coordinating goroutine.
}

type ControlEvent interface{}

type ControlEventRegister struct {
	PlayerId  string
	ReplyChan chan chan ServerEvent
}

type ControlEventUnregister struct {
	PlayerId string
}

func (g *Game) addEventListener(playerId string) chan ServerEvent {
	ch := make(chan chan ServerEvent)
	g.ControlChan <- ControlEventRegister{PlayerId: playerId, ReplyChan: ch}
	return <-ch
}

func (g *Game) removeEventListener(playerId string) {
	g.ControlChan <- ControlEventUnregister{PlayerId: playerId}
}

func generatePlayerId() string {
	p := make([]byte, 16)
	crand.Read(p)
	return hex.EncodeToString(p)
}

func generateGameId() string {
	var alphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	var b strings.Builder
	for i := 0; i < 6; i++ {
		b.WriteRune(alphabet[rand.Intn(len(alphabet))])
	}
	return b.String()
}

func NewGame(id string) *Game {
	return &Game{
		Id:          id,
		ControlChan: make(chan ControlEvent),
	}
}

func startNewGame() (*Game, error) {
	// Try a few times to find an unused game Id, else give up.
	// (I don't like forever loops... 100 attempts is plenty.)
	var game *Game
	for i := 0; i < 100; i++ {
		id := generateGameId()
		ongoingGamesMut.Lock()
		if _, ok := ongoingGames[id]; !ok {
			game = NewGame(id)
			ongoingGames[id] = game
		}
		ongoingGamesMut.Unlock()
		if game != nil {
			go func() {
				eventListeners := make(map[string]chan ServerEvent)
				// A background goroutine will send events to all listeners of the game.
				for {
					tick := time.After(1 * time.Second)
					select {
					case ce := <-game.ControlChan:
						switch e := ce.(type) {
						case ControlEventRegister:
							if ch, ok := eventListeners[e.PlayerId]; ok {
								e.ReplyChan <- ch
							} else {
								ch := make(chan ServerEvent)
								eventListeners[e.PlayerId] = ch
								e.ReplyChan <- ch
							}
						case ControlEventUnregister:
							delete(eventListeners, e.PlayerId)
							if len(eventListeners) == 0 {
								// No more listeners left: end the game.
								log.Printf("Game %s has no listeners left. Finishing.", game.Id)
								return
							}
						}
					case <-tick:
						t := time.Now().Format(time.RFC3339)
						for playerId, ch := range eventListeners {
							ch <- ServerEvent{
								Timestamp:    t,
								DebugMessage: playerId,
							}
						}
					}
				}
			}()
			return game, nil
		}
	}
	return nil, fmt.Errorf("Cannot start a new game")
}

func startNewGameHandler(w http.ResponseWriter, r *http.Request) {
	game, err := startNewGame()
	if err != nil {
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
	}
	http.Redirect(w, r, fmt.Sprintf("%s/%s", r.URL.Path, game.Id), http.StatusSeeOther)
}

func lookupGame(id string) *Game {
	ongoingGamesMut.Lock()
	defer ongoingGamesMut.Unlock()
	return ongoingGames[id]
}

func gameHandler(w http.ResponseWriter, r *http.Request) {
	_, err := r.Cookie("playerId")
	if err != nil {
		playerId := generatePlayerId()
		cookie := &http.Cookie{
			Name:     "playerId",
			Value:    playerId,
			Path:     "/hexz",
			MaxAge:   24 * 60 * 60,
			HttpOnly: true,
			Secure:   false, // also allow plain http
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, cookie)
	}
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
	Timestamp    string `json:"timestamp"`
	DebugMessage string `json:"debugMessage"`
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("Incoming SSE request: ", r.URL.Path)
	// We expect a cookie to identify the player.
	cookie, err := r.Cookie("playerId")
	if err != nil {
		log.Printf("SSE request without cookie from %s", r.RemoteAddr)
		http.Error(w, "Missing playerId cookie", http.StatusBadRequest)
		return
	}
	playerId := cookie.Value
	// Look up game from URL path.
	pathSegs := strings.Split(r.URL.Path, "/")
	gameId := ""
	l := len(pathSegs)
	if l > 0 {
		gameId = pathSegs[l-1]
	}
	game := lookupGame(gameId)
	if game == nil {
		log.Printf("SSE request for invalid game %s", gameId)
		http.Error(w, fmt.Sprintf("Game %s not found", gameId), http.StatusNotFound)
		return
	}
	serverEventChan := game.addEventListener(playerId)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	for {
		select {
		case ev := <-serverEventChan:
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
			game.removeEventListener(playerId)
			return
		}
	}
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("Ignoring request for path: ", r.URL.Path, r.URL.RawQuery)
}

func main() {
	flag.Parse()
	// Make sure we have access to the game HTML file.
	if _, err := readGameHtml(); err != nil {
		log.Fatal("Cannot load game HTML: ", err)
	}
	http.HandleFunc("/hexz/move", moveHandler)
	http.HandleFunc("/hexz/sse/", sseHandler)
	http.HandleFunc("/hexz", startNewGameHandler)
	http.HandleFunc("/hexz/", gameHandler)
	http.HandleFunc("/", defaultHandler)

	addr := fmt.Sprintf("%s:%d", *serverAddress, *serverPort)
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
