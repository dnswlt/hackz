package hexz

const (
	numFieldsFirstRow = 10
	numBoardRows      = 11
)

type GameType string

const (
	gameTypeClassic  GameType = "classic"
	gameTypeFreeform GameType = "freeform"
)

func validGameType(gameType string) bool {
	return gameType == string(gameTypeClassic) || gameType == string(gameTypeFreeform)
}

type GameEngineMove struct {
	playerNum int
	row       int
	col       int
	cellType  CellType
}

type GameEngine interface {
	Init()
	Start()
	InitialResources() ResourceInfo
	NumPlayers() int
	Reset()
	MakeMove(move GameEngineMove) bool
	Board() *Board
	IsDone() bool
	Winner() (playerNum int) // Results are only meaningful if IsDone() is true. 0 for draw.
}

// Dispatches on the gameType to create a corresponding GameEngine.
// The returned GameEngine is initialized and ready to play.
func NewGameEngine(gameType GameType) GameEngine {
	var ge GameEngine
	switch gameType {
	case gameTypeClassic:
		ge = &GameEngineClassic{}
	case gameTypeFreeform:
		ge = &GameEngineFreeform{}
	default:
		panic("Unconsidered game type: " + gameType)
	}
	ge.Init()
	return ge
}

//
// The "classic" hexz game
//

type GameEngineClassic struct {
	board *Board
}

func (g *GameEngineClassic) Board() *Board { return g.board }
func (g *GameEngineClassic) Init() {
	g.board = InitBoard(g)
}
func (g *GameEngineClassic) Start() {
	g.board.State = Running
}

func (g *GameEngineClassic) Reset() {
	g.Init()
}

func (g *GameEngineClassic) NumPlayers() int {
	return 2
}

func (g *GameEngineClassic) InitialResources() ResourceInfo {
	return ResourceInfo{
		NumPieces: map[CellType]int{
			cellNormal: -1, // unlimited
			cellFire:   1,
			cellFlag:   1,
			cellPest:   1,
			cellDeath:  1,
		},
	}
}

func (g *GameEngineClassic) IsDone() bool { return g.board.State == Finished }

func (g *GameEngineClassic) Winner() (playerNum int) {
	if !g.IsDone() {
		return 0
	}
	maxIdx := -1
	maxScore := -1
	uniq := false
	for i, s := range g.board.Score {
		if s > maxScore {
			maxScore = s
			maxIdx = i
			uniq = true
		} else if s == maxScore {
			uniq = false
		}
	}
	if uniq {
		return maxIdx
	}
	return 0
}

func (f *Field) occupied() bool {
	return f.Type == cellDead || f.Owner > 0
}

func (g *GameEngineClassic) recomputeScoreAndState() {
	b := g.board
	s := []int{0, 0}
	openCells := 0
	for _, row := range b.Fields {
		for _, fld := range row {
			if fld.Owner > 0 && !fld.Hidden {
				s[fld.Owner-1]++
			}
			if !fld.occupied() || fld.Hidden {
				// Don't finish the game until all cells are owned and not hidden, or dead.
				openCells++
			}
		}
	}
	b.Score = s
	if openCells == 0 {
		// No more inconclusive cells: game is finished
		b.State = Finished
	}
}

type idx struct {
	r, c int
}

func (b *Board) valid(x idx) bool {
	return x.r >= 0 && x.r < len(b.Fields) && x.c >= 0 && x.c < len(b.Fields[x.r])
}

// Populates ns with valid indices of all neighbor cells. Returns the number of neighbor cells.
// ns must have enough capacity to hold all neighbors. You should pass in a [6]idx slice.
func (b *Board) neighbors(x idx, ns []idx) int {
	shift := x.r & 1 // Depending on the row, neighbors below and above are shifted.
	k := 0
	ns[k] = idx{x.r, x.c + 1}
	if b.valid(ns[k]) {
		k++
	}
	ns[k] = idx{x.r - 1, x.c + shift}
	if b.valid(ns[k]) {
		k++
	}
	ns[k] = idx{x.r - 1, x.c - 1 + shift}
	if b.valid(ns[k]) {
		k++
	}
	ns[k] = idx{x.r, x.c - 1}
	if b.valid(ns[k]) {
		k++
	}
	ns[k] = idx{x.r + 1, x.c - 1 + shift}
	if b.valid(ns[k]) {
		k++
	}
	ns[k] = idx{x.r + 1, x.c + shift}
	if b.valid(ns[k]) {
		k++
	}
	return k
}

func floodFill(b *Board, x idx, cb func(idx) bool) {
	var ns [6]idx
	if !cb(x) {
		return
	}
	n := b.neighbors(x, ns[:])
	for i := 0; i < n; i++ {
		floodFill(b, ns[i], cb)
	}
}

func (c CellType) lifetime() int {
	switch c {
	case cellFire, cellDead, cellDeath:
		return 1
	case cellPest:
		return 3
	}
	return -1 // Live forever
}

func occupyFields(b *Board, playerNum, r, c int, ct CellType) int {
	// Create a board-shaped 2d array that indicates which neighboring cell of (i, j)
	// it shares the free area with.
	// Then find the smallest of these areas and occupy every free cell in it.
	ms := make([][]int, len(b.Fields))
	for k := 0; k < len(ms); k++ {
		ms[k] = make([]int, len(b.Fields[k]))
		for m := 0; m < len(ms[k]); m++ {
			ms[k][m] = -1
		}
	}
	f := &b.Fields[r][c]
	f.Owner = playerNum
	f.Type = ct
	f.Hidden = true
	f.Lifetime = ct.lifetime()
	var areas [6]struct {
		size      int    // Number of free cells in the area
		flags     [2]int // Number of flags along the boundary
		deadCells int    // Number of dead cells along the boundary
	}
	// If the current move sets a flag, this flag counts in all directions.
	if ct == cellFlag {
		for i := 0; i < len(areas); i++ {
			areas[i].flags[playerNum-1]++
		}
	}
	// Flood fill starting from each of (r, c)'s neighbors.
	var ns [6]idx
	n := b.neighbors(idx{r, c}, ns[:])
	for k := 0; k < n; k++ {
		if ms[ns[k].r][ns[k].c] != -1 {
			// k's area is the same area as k-c for some c.
			continue
		}
		floodFill(b, ns[k], func(x idx) bool {
			if ms[x.r][x.c] == k {
				// Already seen in this iteration.
				return false
			}
			f := &b.Fields[x.r][x.c]
			if f.occupied() {
				// Occupied fields act as boundaries.
				if f.Type == cellFlag {
					areas[k].flags[f.Owner-1]++
				} else if f.Type == cellDead {
					areas[k].deadCells++
				}
				// Mark as seen to avoid revisiting boundaries.
				ms[x.r][x.c] = k
				return false
			}
			// Mark as seen and update area size.
			ms[x.r][x.c] = k
			areas[k].size++
			return true
		})
	}
	// If there is more than one area, we know we introduced a split, since the areas
	// would have been connected by the previously free cell (r, c).
	numOccupiedFields := 1
	numAreas := 0
	minK := -1
	// Count the number of separated areas and find the smallest one.
	for k := 0; k < n; k++ {
		if areas[k].size > 0 && areas[k].deadCells == 0 {
			numAreas++
			if minK == -1 || areas[minK].size > areas[k].size {
				minK = k
			}
		}
	}
	if numAreas > 1 {
		// Now assign fields to player with most flags, or to current player on a tie.
		occupator := playerNum
		if areas[minK].flags[2-playerNum] > areas[minK].flags[playerNum-1] {
			occupator = 3 - playerNum // The other player has more flags
		}
		for r := 0; r < len(b.Fields); r++ {
			for c := 0; c < len(b.Fields[r]); c++ {
				f := &b.Fields[r][c]
				if ms[r][c] == minK && !f.occupied() {
					numOccupiedFields++
					f.Owner = occupator
					f.Lifetime = cellNormal.lifetime()
				}
			}
		}
	}
	return numOccupiedFields
}

func applyFireEffect(b *Board, r, c int) {
	var ns [6]idx
	n := b.neighbors(idx{r, c}, ns[:])
	for i := 0; i < n; i++ {
		f := &b.Fields[ns[i].r][ns[i].c]
		f.Owner = 0
		f.Type = cellDead
		f.Hidden = false
		f.Lifetime = cellDead.lifetime()
	}
}

func applyPestEffect(b *Board) {
	var ns [6]idx
	for r := 0; r < len(b.Fields); r++ {
		for c := 0; c < len(b.Fields[r]); c++ {
			// A pest cell does not propagate in its first round.
			if b.Fields[r][c].Type == cellPest && b.Fields[r][c].Lifetime < cellPest.lifetime() {
				n := b.neighbors(idx{r, c}, ns[:])
				for i := 0; i < n; i++ {
					f := &b.Fields[ns[i].r][ns[i].c]
					if f.Owner > 0 && f.Owner != b.Fields[r][c].Owner && f.Type == cellNormal {
						// Pest only affects the opponent's normal cells.
						f.Owner = b.Fields[r][c].Owner
						f.Type = cellPest
						f.Lifetime = cellPest.lifetime()
					}
				}
			}
		}
	}
}

func (g *GameEngineClassic) MakeMove(m GameEngineMove) bool {
	board := g.board
	turn := board.Turn
	if m.playerNum != turn {
		// Only allow moves by players whose turn it is.
		return false
	}
	if !board.valid(idx{m.row, m.col}) || m.cellType == cellDead {
		// Invalid move request.
		return false
	}
	if m.cellType != cellNormal && board.Resources[turn-1].NumPieces[m.cellType] == 0 {
		// No pieces left of requested type
		return false
	}
	numOccupiedFields := 0
	revealBoard := m.cellType != cellNormal && m.cellType != cellFlag
	if board.Fields[m.row][m.col].occupied() {
		if board.Fields[m.row][m.col].Hidden && board.Fields[m.row][m.col].Owner == (3-turn) {
			// Conflicting hidden moves. Leads to dead cell.
			board.Move++
			f := &board.Fields[m.row][m.col]
			f.Owner = 0
			f.Type = cellDead
			f.Lifetime = cellDead.lifetime()
			revealBoard = true
		} else if m.cellType == cellDeath {
			// Death cell can be placed anywhere and will "kill" whatever was there before.
			f := &board.Fields[m.row][m.col]
			f.Owner = turn
			f.Type = cellDeath
			f.Hidden = false
			f.Lifetime = cellDeath.lifetime()
		} else {
			// Cannot make move on already occupied field.
			return false
		}
	} else {
		// Free cell: occupy it.
		board.Move++
		f := &board.Fields[m.row][m.col]
		if m.cellType == cellFire {
			// Fire cells take effect immediately.
			f.Owner = turn
			f.Type = m.cellType
			f.Lifetime = cellFire.lifetime()
			applyFireEffect(board, m.row, m.col)
		} else {
			numOccupiedFields = occupyFields(board, turn, m.row, m.col, m.cellType)
		}
	}
	if m.cellType != cellNormal {
		board.Resources[turn-1].NumPieces[m.cellType]--
	}
	// Update turn.
	board.Turn++
	if board.Turn > 2 {
		board.Turn = 1
	}
	if numOccupiedFields > 1 || board.Move-board.LastRevealed == 4 || revealBoard {
		// Reveal hidden moves.
		for r := 0; r < len(board.Fields); r++ {
			for c := 0; c < len(board.Fields[r]); c++ {
				f := &board.Fields[r][c]
				f.Hidden = false
			}
		}
		applyPestEffect(board)
		// Clean up old special cells.
		for r := 0; r < len(board.Fields); r++ {
			for c := 0; c < len(board.Fields[r]); c++ {
				f := &board.Fields[r][c]
				if f.occupied() && f.Lifetime == 0 {
					f.Owner = 0
					f.Hidden = false
					f.Type = cellNormal
					f.Lifetime = cellNormal.lifetime()
				}
				if f.Lifetime > 0 {
					f.Lifetime--
				}
			}
		}

		board.LastRevealed = board.Move
	}
	g.recomputeScoreAndState()
	return true
}

//
// The freeform single-player hexz game.
//

type GameEngineFreeform struct {
	board *Board
}

func (g *GameEngineFreeform) Board() *Board { return g.board }

func (g *GameEngineFreeform) Init() {
	g.board = InitBoard(g)
}
func (g *GameEngineFreeform) Start() {
	g.board.State = Running
}
func (g *GameEngineFreeform) NumPlayers() int {
	return 1
}

func (g *GameEngineFreeform) InitialResources() ResourceInfo {
	return ResourceInfo{
		NumPieces: map[CellType]int{
			cellNormal: -1, // unlimited
			cellFire:   -1,
			cellFlag:   -1,
			cellPest:   -1,
			cellDeath:  -1,
		},
	}
}

func (g *GameEngineFreeform) Reset()       { g.Init() }
func (g *GameEngineFreeform) IsDone() bool { return false }
func (g *GameEngineFreeform) Winner() (playerNum int) {
	return 0 // No one ever wins here.
}

// Creates a new, empty 2d field array.
func makeFields() [][]Field {
	const numFields = numFieldsFirstRow*((numBoardRows+1)/2) + (numFieldsFirstRow-1)*(numBoardRows/2)
	arr := make([]Field, numFields)
	fields := make([][]Field, numBoardRows)
	start := 0
	for i := 0; i < len(fields); i++ {
		end := start + numFieldsFirstRow - i%2
		fields[i] = arr[start:end]
		start = end
	}
	return fields
}

func InitBoard(g GameEngine) *Board {
	b := &Board{
		Turn:   1, // Player 1 begins
		Fields: makeFields(),
		State:  Initial,
	}
	numPlayers := g.NumPlayers()
	b.Score = make([]int, numPlayers)
	b.Resources = make([]ResourceInfo, numPlayers)
	for i := 0; i < numPlayers; i++ {
		b.Resources[i] = g.InitialResources()
	}
	return b
}

func (g *GameEngineFreeform) MakeMove(m GameEngineMove) bool {
	board := g.board
	if !board.valid(idx{m.row, m.col}) {
		// Invalid move request.
		return false
	}
	board.Move++
	f := &board.Fields[m.row][m.col]
	f.Owner = board.Turn
	f.Type = m.cellType
	board.Turn++
	if board.Turn > 2 {
		board.Turn = 1
	}
	return true
}
