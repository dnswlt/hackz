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

const (
	numFieldsFirstRow = 10
	numBoardRows      = 11
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

type ControlEvent interface {
	controlEventImpl() // Interface marker function
}

type ControlEventRegister struct {
	PlayerId  string
	ReplyChan chan chan ServerEvent
}

type ControlEventUnregister struct {
	PlayerId string
}

type ControlEventMove struct {
	PlayerId string
	Row      int
	Col      int
}

func (e ControlEventRegister) controlEventImpl()   {}
func (e ControlEventUnregister) controlEventImpl() {}
func (e ControlEventMove) controlEventImpl()       {}

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

func NewBoard() *Board {
	fields := make([][]Field, numBoardRows)
	for i := 0; i < len(fields); i++ {
		n := numFieldsFirstRow - i%2
		fields[i] = make([]Field, n)
	}
	return &Board{
		Turn:   1, // Player 1 begins
		Fields: fields,
	}
}

// Controller function for a running game. To be executed by a dedicated goroutine.
func gameMaster(game *Game) {
	const numPlayers = 2
	eventListeners := make(map[string]chan ServerEvent)
	board := NewBoard()
	players := make(map[string]int)
	broadcast := func(e ServerEvent) {
		e.Timestamp = time.Now().Format(time.RFC3339)
		for _, ch := range eventListeners {
			ch <- e
		}
	}
	for {
		tick := time.After(5 * time.Second)
		select {
		case ce := <-game.ControlChan:
			switch e := ce.(type) {
			case ControlEventRegister:
				if ch, ok := eventListeners[e.PlayerId]; ok {
					e.ReplyChan <- ch
				} else {
					// The first two participants in the game are players.
					// Anyone arriving later will be a spectator.
					if len(players) < numPlayers {
						players[e.PlayerId] = len(players) + 1
					}
					ch := make(chan ServerEvent)
					eventListeners[e.PlayerId] = ch
					e.ReplyChan <- ch
				}
			case ControlEventUnregister:
				delete(eventListeners, e.PlayerId)
				delete(players, e.PlayerId)
				if len(eventListeners) == 0 {
					// No more listeners left: end the game.
					log.Printf("Game %s has no listeners left. Finishing.", game.Id)
					return
				}
			case ControlEventMove:
				playerNum, ok := players[e.PlayerId]
				if !ok || board.Turn != playerNum {
					// Only allow moves by players whose turn it is.
					break
				}
				if e.Row < 0 || e.Row >= len(board.Fields) || e.Col < 0 || e.Col >= len(board.Fields[e.Row]) {
					break
				}
				board.Fields[e.Row][e.Col].Value = 1
				board.Fields[e.Row][e.Col].Owner = playerNum
				board.Turn++
				if board.Turn > numPlayers {
					board.Turn = 1
				}
				broadcast(ServerEvent{Board: board})
			}
		case <-tick:
			broadcast(ServerEvent{DebugMessage: "ping"})
		}
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
			go gameMaster(game)
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
	Row int `json:"row"`
	Col int `json:"col"`
}

func moveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusBadRequest)
		return
	}
	cookie, err := r.Cookie("playerId")
	if err != nil {
		http.Error(w, "Missing playerId cookie", http.StatusBadRequest)
		return
	}
	playerId := cookie.Value
	dec := json.NewDecoder(r.Body)
	var req MoveRequest
	if err := dec.Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	gameId := gameIdFromPath(r.URL.Path)
	game := lookupGame(gameId)
	if game == nil {
		http.Error(w, fmt.Sprintf("No game with ID %q", gameId), http.StatusNotFound)
		return
	}
	log.Printf("Received move request from %s: (%d, %d)", playerId, req.Row, req.Col)
	game.ControlChan <- ControlEventMove{PlayerId: playerId, Row: req.Row, Col: req.Col}
}

type Field struct {
	Value int `json:"value"`
	Owner int `json:"owner"` // Player number owning this field.
}

type Board struct {
	Turn   int       `json:"turn"`
	Fields [][]Field `json:"fields"`
}

type ServerEvent struct {
	Timestamp    string `json:"timestamp"`
	Board        *Board `json:"board"`
	DebugMessage string `json:"debugMessage"`
}

func gameIdFromPath(path string) string {
	// Look up game from URL path.
	pathSegs := strings.Split(path, "/")
	gameId := ""
	l := len(pathSegs)
	if l > 0 {
		gameId = pathSegs[l-1]
	}
	return gameId
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
	gameId := gameIdFromPath(r.URL.Path)
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
	http.HandleFunc("/hexz/move/", moveHandler)
	http.HandleFunc("/hexz/sse/", sseHandler)
	http.HandleFunc("/hexz", startNewGameHandler)
	http.HandleFunc("/hexz/", gameHandler)
	http.HandleFunc("/", defaultHandler)

	addr := fmt.Sprintf("%s:%d", *serverAddress, *serverPort)
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
