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
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	gameHtmlFile       = flag.String("html", "game.html", "Path to game HTML file")
	serverAddress      = flag.String("address", "", "Address on which to listen")
	serverPort         = flag.Int("port", 8084, "Port on which to listen")
	gameGcDelaySeconds = flag.Int("gcdelay", 5,
		"Seconds to wait before deleting a disconnected player from a game")

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
	Started     time.Time
	ControlChan chan ControlEvent // The channel to communicate with the game coordinating goroutine.
}

// JSON for server responses.
type Field struct {
	Value int `json:"value"`
	Owner int `json:"owner"` // Player number owning this field.
}

type Board struct {
	Turn   int       `json:"turn"`
	Fields [][]Field `json:"fields"`
}

type ServerEvent struct {
	Timestamp    string   `json:"timestamp"`
	Board        *Board   `json:"board"`
	Role         int      `json:"role"` // 0: spectator, 1, 2: players
	DebugMessage string   `json:"debugMessage"`
	ActiveGames  []string `json:"activeGames"`
}

// JSON for incoming requests from UI clients.
type MoveRequest struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// Control events are sent to the game master goroutine.
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

// Generates a random 128-bit hex string representing a player ID.
func generatePlayerId() string {
	p := make([]byte, 16)
	crand.Read(p)
	return hex.EncodeToString(p)
}

// Generates a 6-letter game ID.
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
		Started:     time.Now(),
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

// Looks up the game ID from the URL path.
func gameIdFromPath(path string) string {
	pathSegs := strings.Split(path, "/")
	gameId := ""
	l := len(pathSegs)
	if l > 0 {
		gameId = pathSegs[l-1]
	}
	return gameId
}

// Controller function for a running game. To be executed by a dedicated goroutine.
func gameMaster(game *Game) {
	const numPlayers = 2
	gcTimeout := time.Duration(*gameGcDelaySeconds) * time.Second
	board := NewBoard()
	eventListeners := make(map[string]chan ServerEvent)
	players := make(map[string]int) // playerId => player number (1, 2)
	playerGcCancel := make(map[string]chan bool)
	gcChan := make(chan string)
	broadcast := func(e ServerEvent) {
		e.Timestamp = time.Now().Format(time.RFC3339)
		for _, ch := range eventListeners {
			ch <- e
		}
	}
	singlecast := func(playerId string, e ServerEvent) {
		if ch, ok := eventListeners[playerId]; ok {
			e.Timestamp = time.Now().Format(time.RFC3339)
			ch <- e
		}
	}
	for {
		tick := time.After(5 * time.Second)
		select {
		case ce := <-game.ControlChan:
			switch e := ce.(type) {
			case ControlEventRegister:
				if _, ok := players[e.PlayerId]; ok {
					// Player reconnected. Cancel its GC.
					if cancel, ok := playerGcCancel[e.PlayerId]; ok {
						log.Printf("Player %s reconnected. Cancelling GC.", e.PlayerId)
						cancel <- true
						delete(playerGcCancel, e.PlayerId)
					}
				} else {
					// The first two participants in the game are players.
					// Anyone arriving later will be a spectator.
					if len(players) < numPlayers {
						players[e.PlayerId] = len(players) + 1
					}
				}
				ch := make(chan ServerEvent)
				eventListeners[e.PlayerId] = ch
				e.ReplyChan <- ch
				// Send board and player role initially so client can display the UI.
				singlecast(e.PlayerId, ServerEvent{Board: board, ActiveGames: listRecentGames(5), Role: players[e.PlayerId]})
			case ControlEventUnregister:
				delete(eventListeners, e.PlayerId)
				if _, ok := playerGcCancel[e.PlayerId]; ok {
					// A repeated unregister should not happen. If it does, we ignore
					// it and just wait for the existing GC "callback" to happen.
					break
				}
				// Remove player after timeout. Don't remove them immediately as they might
				// just be reloading their page and rejoin soon.
				cancelChan := make(chan bool, 1)
				playerGcCancel[e.PlayerId] = cancelChan
				go func(playerId string) {
					t := time.After(gcTimeout)
					select {
					case <-t:
						gcChan <- playerId
					case <-cancelChan:
					}
				}(e.PlayerId)
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
			broadcast(ServerEvent{ActiveGames: listRecentGames(5), DebugMessage: "ping"})
		case playerId := <-gcChan:
			if _, ok := playerGcCancel[playerId]; !ok {
				// Ignore zombie GC message. Player has already reconnected.
				log.Printf("Ignoring GC message for player %s in game %s", playerId, game.Id)
			}
			log.Printf("Player %s has left game %s", playerId, game.Id)
			delete(eventListeners, playerId)
			delete(players, playerId)
			delete(playerGcCancel, playerId)
			if len(players) == 0 {
				// No more players left: end the game.
				log.Printf("Game %s has no players left. Finishing.", game.Id)
				deleteGame(game.Id)
				return
			}
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

func deleteGame(id string) {
	ongoingGamesMut.Lock()
	defer ongoingGamesMut.Unlock()
	delete(ongoingGames, id)
}

func lookupGame(id string) *Game {
	ongoingGamesMut.Lock()
	defer ongoingGamesMut.Unlock()
	return ongoingGames[id]
}

func listRecentGames(limit int) []string {
	ongoingGamesMut.Lock()
	games := []*Game{}
	for _, g := range ongoingGames {
		games = append(games, g)
	}
	ongoingGamesMut.Unlock()
	sort.Slice(games, func(i, j int) bool {
		return games[i].Started.After(games[j].Started)
	})
	n := limit
	if limit > len(games) {
		n = len(games)
	}
	ids := make([]string, n)
	for i, g := range games[:n] {
		ids[i] = g.Id
	}
	return ids
}

func handleHexz(w http.ResponseWriter, r *http.Request) {
	// For now, immediately redirect to a new game.
	game, err := startNewGame()
	if err != nil {
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
	}
	http.Redirect(w, r, fmt.Sprintf("%s/%s", r.URL.Path, game.Id), http.StatusSeeOther)
}

func handleMove(w http.ResponseWriter, r *http.Request) {
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

func handleSse(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, fmt.Sprintf("Game %s does not exist", gameId), http.StatusNotFound)
		return
	}
	serverEventChan := game.addEventListener(playerId)
	// Headers to establish server-sent events (SSE) communication.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	for {
		select {
		case ev := <-serverEventChan:
			// Send ServerEvent JSON on SSE connection.
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

func handleGame(w http.ResponseWriter, r *http.Request) {
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

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	// Ignore
	log.Print("Ignoring request for path: ", r.URL.Path, r.URL.RawQuery)
}

func main() {
	flag.Parse()
	// Make sure we have access to the game HTML file.
	if _, err := readGameHtml(); err != nil {
		log.Fatal("Cannot load game HTML: ", err)
	}
	http.HandleFunc("/hexz/move/", handleMove)
	http.HandleFunc("/hexz/sse/", handleSse)
	http.HandleFunc("/hexz", handleHexz)
	http.HandleFunc("/hexz/", handleGame)
	http.HandleFunc("/", defaultHandler)

	addr := fmt.Sprintf("%s:%d", *serverAddress, *serverPort)
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
