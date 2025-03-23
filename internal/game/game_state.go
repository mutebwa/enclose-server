package game

import "time"

const (
	GRID_SIZE       = 64
	MAX_PLAYERS     = 2
	SESSION_TIMEOUT = time.Minute * 15
)

type State struct {
	Grid         [][]Point
	Scores       [2]int
	CurrentTurn  int
	GameOver     bool
	Message      string   
	PlayersReady []bool
	LastMoveAt   time.Time
}

func NewState() State {
	return State{
		Grid:        createGrid(64),
		Scores:      [2]int{0, 0},
		CurrentTurn: 1,
	}
}

type Point struct {
	X      int
	Y      int
	Owner  int // 0: empty, 1: player1, 2: player2
	Sealed bool
}

func createGrid(size int) [][]Point {
	grid := make([][]Point, size)
	for y := range grid {
		grid[y] = make([]Point, size)
		for x := range grid[y] {
			grid[y][x] = Point{X: x, Y: y}
		}
	}
	return grid
}