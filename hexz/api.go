package hexz

import "time"

type GameState string

const (
	Initial  GameState = "initial"
	Running  GameState = "running"
	Finished GameState = "finished"
)

// JSON for server responses.

type ServerEvent struct {
	Timestamp     string     `json:"timestamp"`
	Board         *BoardView `json:"board"`
	Role          int        `json:"role"` // 0: spectator, 1, 2: players
	PlayerNames   []string   `json:"playerNames"`
	Announcements []string   `json:"announcements"`
	DebugMessage  string     `json:"debugMessage"`
	Winner        int        `json:"winner,omitempty"` // Number of the player that wins. 0 if no winner yet or draw.
	LastEvent     bool       `json:"lastEvent"`        // Signals to clients that this is the last event they will receive.
}

// A player's or spectator's view of the board.
// See type Board for the internal representation that holds the complete information.
type BoardView struct {
	Turn        int            `json:"turn"`
	Move        int            `json:"move"`
	Fields      [][]Field      `json:"fields"` // The board's fields.
	PlayerNames []string       `json:"playerNames"`
	Score       []int          `json:"score"` // Depending on the number of players, 1 or 2 elements.
	Resources   []ResourceInfo `json:"resources"`
	State       GameState      `json:"state"`
}

type Field struct {
	Type     CellType `json:"type"`
	Owner    int      `json:"owner"` // Player number owning this field. 0 for unowned fields.
	Hidden   bool     `json:"hidden,omitempty"`
	Value    int      `json:"v"`                 // Some games assign different values to cells.
	Blocked  int      `json:"blocked,omitempty"` // Bitmap indicating which players this field is blocked for.
	lifetime int      // Moves left until this cell gets cleared. -1 means infinity.
	nextVal  [2]int   // If this cell would be occupied, what value would it have? (For Flagz)
}

// Information about the resources each player has left.
type ResourceInfo struct {
	NumPieces map[CellType]int `json:"numPieces"`
}

type CellType int

// Remember to update the known cell types in game.html if you make changes here!
const (
	cellNormal CellType = iota // Empty cells if not owned, otherwise the player's regular cell.
	// Non-player cells.
	cellDead  // A dead cell. Usually generated from a piece placement conflict.
	cellGrass // Introduced in the Flagz game. Cells that can be collected.
	cellRock  // Unowned and similar to a dead cell. Can be used to build static obstacles.
	// Player's special pieces. Update isPlayerPiece() if you make changes.
	cellFire
	cellFlag
	cellPest
	cellDeath // If you add one below cellDeath, update valid().
)

func (c CellType) valid() bool {
	return c >= cellNormal && c <= cellDeath
}

func (c CellType) isPlayerPiece() bool {
	return c == cellNormal || c >= cellFire && c <= cellDead
}

// JSON for incoming requests from UI clients.
type MoveRequest struct {
	Move int      `json:"move"` // Used to discard move requests that do not match the game's current state.
	Row  int      `json:"row"`
	Col  int      `json:"col"`
	Type CellType `json:"type"`
}

type ResetRequest struct {
	Message string `json:"message"`
}

type StatuszCounter struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type StatuszDistribBucket struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"` // exclusive
	Count int64   `json:"count"`
}

type StatuszDistrib struct {
	Name    string                 `json:"name"`
	Buckets []StatuszDistribBucket `json:"buckets"`
}

type StatuszResponse struct {
	Started            time.Time         `json:"started"`
	UptimeSeconds      int               `json:"uptimeSeconds"`
	Uptime             string            `json:"uptime"` // 1h30m3.5s
	NumOngoingGames    int               `json:"numOngoingGames"`
	NumLoggedInPlayers int               `json:"numLoggedInPlayers"`
	Counters           []StatuszCounter  `json:"counters"`
	Distributions      []*StatuszDistrib `json:"distributions"`
}

// Used in responses to list active games (/hexz/gamez).
type GameInfo struct {
	Id       string    `json:"id"`
	Host     string    `json:"host"`
	Started  time.Time `json:"started"`
	GameType GameType  `json:"gameType"`
}
