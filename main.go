package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ===============================
// Constants & Config
// ===============================
const (
	GRID_SIZE       = 64
	MAX_PLAYERS     = 2
	SESSION_TIMEOUT = time.Minute * 15
)

// ===============================
// Types & Interfaces
// ===============================
type Point struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Owner  int `json:"owner"`  // 0: empty, 1: player1, 2: player2
	Sealed bool `json:"sealed"`
}

type GameState struct {
	Grid         [][]Point `json:"grid"`
	Scores       []int     `json:"scores"`       // [P1, P2]
	CurrentTurn  int       `json:"currentTurn"`  // 1 or 2
	GameOver     bool      `json:"gameOver"`
	Message      string    `json:"message"`
	PlayersReady []bool    `json:"playersReady"`
}

type Player struct {
	ID        string          `json:"id"`
	Conn      *websocket.Conn `json:"-"`
	Number    int             `json:"number"` // 1 or 2
	LastSeen  time.Time       `json:"-"`
}

type Game struct {
	ID           string
	Players      map[string]*Player
	State        GameState
	mu           sync.RWMutex
	CreatedAt    time.Time
	ActivityChan chan struct{}
}

// ===============================
// Global State
// ===============================
var (
	games   = make(map[string]*Game)
	gameMu  sync.RWMutex
	upgrader = websocket.Upgrader{
		CheckOrigin:     func(r *http.Request) bool { return true },
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

// ===============================
// Helper Functions
// ===============================
func createEmptyGrid() [][]Point {
	grid := make([][]Point, GRID_SIZE)
	for y := range grid {
		grid[y] = make([]Point, GRID_SIZE)
		for x := range grid[y] {
			grid[y][x] = Point{X: x, Y: y}
		}
	}
	return grid
}

func deepCopyGrid(grid [][]Point) [][]Point {
	newGrid := make([][]Point, len(grid))
	for i := range grid {
		newGrid[i] = make([]Point, len(grid[i]))
		copy(newGrid[i], grid[i])
	}
	return newGrid
}

// Improved enclosure detection (matches frontend logic)
func detectEnclosures(grid [][]Point, currentPlayer int) ([][]Point, bool) {
	newGrid := deepCopyGrid(grid)
	size := len(newGrid)
	visited := make([][]bool, size)
	for i := range visited {
		visited[i] = make([]bool, size)
	}

	// Edge initialization
	queue := make([][2]int, 0)
	for i := 0; i < size; i++ {
		queue = append(queue, [2]int{0, i}, [2]int{size-1, i}, [2]int{i, 0}, [2]int{i, size-1})
	}

	// BFS from edges
	directions := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for len(queue) > 0 {
		cell := queue[0]
		queue = queue[1:]
		x, y := cell[0], cell[1]

		if visited[y][x] || newGrid[y][x].Owner == currentPlayer || newGrid[y][x].Sealed {
			continue
		}

		visited[y][x] = true
		for _, d := range directions {
			nx, ny := x+d[0], y+d[1]
			if nx >= 0 && nx < size && ny >= 0 && ny < size && !visited[ny][nx] {
				queue = append(queue, [2]int{nx, ny})
			}
		}
	}

	// Seal unvisited cells
	enclosed := false
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if !visited[y][x] && !newGrid[y][x].Sealed {
				if newGrid[y][x].Owner != 0 && newGrid[y][x].Owner != currentPlayer {
					enclosed = true
				}
				newGrid[y][x].Sealed = true
			}
		}
	}

	return newGrid, enclosed
}

// ===============================
// Game Management
// ===============================
func createGame() *Game {
	gameID := fmt.Sprintf("game-%d", time.Now().UnixNano())
	return &Game{
		ID:      gameID,
		Players: make(map[string]*Player),
		State: GameState{
			Grid:        createEmptyGrid(),
			Scores:      []int{0, 0},
			CurrentTurn: 1,
			PlayersReady: make([]bool, MAX_PLAYERS),
		},
		CreatedAt:    time.Now(),
		ActivityChan: make(chan struct{}, 1),
	}
}

func findAvailableGame() *Game {
	gameMu.Lock()
	defer gameMu.Unlock()

	for _, game := range games {
		if len(game.Players) < MAX_PLAYERS && time.Since(game.CreatedAt) < SESSION_TIMEOUT {
			return game
		}
	}

	newGame := createGame()
	games[newGame.ID] = newGame
	return newGame
}

// ===============================
// WebSocket Handlers
// ===============================
func (g *Game) broadcastState() {
	g.mu.RLock()
	defer g.mu.RUnlock()

	stateCopy := GameState{
		Grid:        deepCopyGrid(g.State.Grid),
		Scores:      append([]int{}, g.State.Scores...),
		CurrentTurn: g.State.CurrentTurn,
		GameOver:    g.State.GameOver,
		Message:     g.State.Message,
		PlayersReady: append([]bool{}, g.State.PlayersReady...),
	}

	for _, p := range g.Players {
		msg := map[string]interface{}{
			"type":      "gameState",
			"state":     stateCopy,
			"playerId":  p.ID,
			"playerNumber": p.Number,
		}
		if err := p.Conn.WriteJSON(msg); err != nil {
			log.Printf("Error sending state to %s: %v", p.ID, err)
		}
	}
}

func (g *Game) handleMove(playerID string, x, y int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	player, exists := g.Players[playerID]
	if !exists || player.Number != g.State.CurrentTurn || g.State.GameOver {
		return
	}

	if x < 0 || x >= GRID_SIZE || y < 0 || y >= GRID_SIZE || g.State.Grid[y][x].Owner != 0 {
		return
	}

	// Update grid
	newGrid := deepCopyGrid(g.State.Grid)
	newGrid[y][x].Owner = player.Number

	// Detect enclosures
	updatedGrid, enclosed := detectEnclosures(newGrid, player.Number)

	// Update scores
	newScores := append([]int{}, g.State.Scores...)
	if enclosed {
		newScores[player.Number-1] = calculateScore(updatedGrid, player.Number)
	}

	// Check game over
	gameOver := isGameOver(updatedGrid)

	// Update turn
	nextTurn := player.Number
	if !enclosed {
		nextTurn = player.Number%2 + 1
	}

	// Update state
	g.State = GameState{
		Grid:        updatedGrid,
		Scores:      newScores,
		CurrentTurn: nextTurn,
		GameOver:    gameOver,
		PlayersReady: g.State.PlayersReady,
	}

	if gameOver {
		g.State.Message = getGameResultMessage(updatedGrid)
	}

	g.broadcastState()
}

func calculateScore(grid [][]Point, player int) int {
	score := 0
	for y := range grid {
		for x := range grid[y] {
			if grid[y][x].Sealed {
				if grid[y][x].Owner != 0 && grid[y][x].Owner != player {
					score++
				}
			}
		}
	}
	return score
}

func isGameOver(grid [][]Point) bool {
	for y := range grid {
		for x := range grid[y] {
			if grid[y][x].Owner == 0 {
				return false
			}
		}
	}
	return true
}

func getGameResultMessage(grid [][]Point) string {
	p1, p2 := 0, 0
	for y := range grid {
		for x := range grid[y] {
			switch grid[y][x].Owner {
			case 1: p1++
			case 2: p2++
			}
		}
	}
	
	switch {
	case p1 > p2: return "Player 1 Wins!"
	case p2 > p1: return "Player 2 Wins!"
	default: return "It's a Tie!"
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	game := findAvailableGame()
	player := &Player{
		ID:       fmt.Sprintf("player-%d", time.Now().UnixNano()),
		Conn:     conn,
		Number:   len(game.Players) + 1,
		LastSeen: time.Now(),
	}

	game.mu.Lock()
	game.Players[player.ID] = player
	game.mu.Unlock()

	// Send initial connection info
	conn.WriteJSON(map[string]interface{}{
		"type":         "connection",
		"gameId":      game.ID,
		"playerId":    player.ID,
		"playerNumber": player.Number,
	})

	game.broadcastState()

	// Message handling
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		game.ActivityChan <- struct{}{}

		switch msg["type"] {
		case "move":
			x, _ := msg["x"].(float64)
			y, _ := msg["y"].(float64)
			playerID, _ := msg["playerId"].(string)
			game.handleMove(playerID, int(x), int(y))

		case "ready":
			playerID, _ := msg["playerId"].(string)
			game.mu.Lock()
			if p, exists := game.Players[playerID]; exists {
				game.State.PlayersReady[p.Number-1] = true
				if game.State.PlayersReady[0] && game.State.PlayersReady[1] {
					game.State.Message = "Game Starting!"
					game.State.CurrentTurn = 1
				}
			}
			game.mu.Unlock()
			game.broadcastState()

		case "reset":
			game.mu.Lock()
			game.State = GameState{
				Grid:         createEmptyGrid(),
				Scores:       []int{0, 0},
				CurrentTurn:  1,
				PlayersReady: make([]bool, MAX_PLAYERS),
			}
			game.mu.Unlock()
			game.broadcastState()
		}
	}

	// Cleanup
	game.mu.Lock()
	delete(game.Players, player.ID)
	remaining := len(game.Players)
	game.mu.Unlock()

	if remaining == 0 {
		gameMu.Lock()
		delete(games, game.ID)
		gameMu.Unlock()
	}
}

func main() {
	// Start cleanup goroutine
	go func() {
		for range time.Tick(time.Minute) {
			gameMu.Lock()
			for id, game := range games {
				if time.Since(game.CreatedAt) > SESSION_TIMEOUT || len(game.Players) == 0 {
					delete(games, id)
				}
			}
			gameMu.Unlock()
		}
	}()

	http.HandleFunc("/ws", handleWebSocket)
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}