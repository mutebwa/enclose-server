package game

import "time"

func ProcessMove(g *Game, x, y, playerNumber int) {
	// Update grid
	g.State.Grid[y][x].Owner = playerNumber

	// Detect enclosures
	newGrid, enclosed := detectEnclosures(g.State.Grid, playerNumber)
	
	// Update scores if enclosed
	if enclosed {
		g.State.Scores[playerNumber-1] = calculateScore(newGrid, playerNumber)
	}

	// Update game state
	g.State.Grid = newGrid
	g.State.LastMoveAt = time.Now()
	
	// Check game over
	if isGameOver(newGrid) {
		g.State.GameOver = true
		g.State.Message = getGameResult(newGrid)
	} else {
		// Update turn if no enclosure
		if !enclosed {
			g.State.CurrentTurn = playerNumber%2 + 1
		}
	}
}

func calculateScore(grid [][]Point, player int) int {
	score := 0
	for y := range grid {
		for x := range grid[y] {
			if grid[y][x].Sealed && grid[y][x].Owner != 0 && grid[y][x].Owner != player {
				score++
			}
		}
	}
	return score
}

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
		queue = append(queue, [2]int{0, i}, [2]int{size - 1, i}, [2]int{i, 0}, [2]int{i, size - 1})
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

func deepCopyGrid(grid [][]Point) [][]Point {
	newGrid := make([][]Point, len(grid))
	for i := range grid {
		newGrid[i] = make([]Point, len(grid[i]))
		copy(newGrid[i], grid[i])
	}
	return newGrid
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

func getGameResult(grid [][]Point) string {
	p1, p2 := 0, 0
	for y := range grid {
		for x := range grid[y] {
			switch grid[y][x].Owner {
			case 1:
				p1++
			case 2:
				p2++
			}
		}
	}

	switch {
	case p1 > p2:
		return "Player 1 Wins!"
	case p2 > p1:
		return "Player 2 Wins!"
	default:
		return "It's a Tie!"
	}
}