package game

import (
	"enclose/internal/player"
	"sync"
	"time"
)

type Player = player.Player

type Game struct {
	ID        string
	Players   map[string]*Player
	State     State
	Mu        sync.RWMutex
	createdAt time.Time
	stopChan  chan struct{}
}

func NewGame() *Game {
	g := &Game{
		ID:        generateID(),
		Players:   make(map[string]*Player),
		State:     NewState(),
		createdAt: time.Now(),
		stopChan:  make(chan struct{}),
	}
	go g.maintainConnections()
	return g
}

func (g *Game) maintainConnections() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			g.CheckInactivePlayers()
		case <-g.stopChan:
			return
		}
	}
}

func (g *Game) CheckInactivePlayers() {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	for id, p := range g.Players {
		if time.Since(p.LastSeen) > 15*time.Second {
			delete(g.Players, id)
		}
	}
}

func (g *Game) IsExpired() bool {
	return time.Since(g.createdAt) > 30*time.Minute
}

func (g *Game) IsEmpty() bool {
	return len(g.Players) == 0
}

func (g *Game) Stop() {
	close(g.stopChan)
}

func generateID() string {
	return "game-" + time.Now().Format("20060102150405")
}

// internal/game/game.go

func (g *Game) IsFull() bool {
	g.Mu.RLock()
	defer g.Mu.RUnlock()
	return len(g.Players) >= MAX_PLAYERS
}

func (g *Game) AllPlayersReady() bool {
	g.Mu.RLock()
	defer g.Mu.RUnlock()
	for _, ready := range g.State.PlayersReady {
		if !ready {
			return false
		}
	}
	return true
}

func (g *Game) Start() {
	g.State.CurrentTurn = 1
	g.State.GameOver = false
	g.State.Message = "Game started!"
}

func (g *Game) Reset() {
	g.State = NewState()
}

func (g *Game) BroadcastState() {
	for _, p := range g.Players {
		p.SendJSON(map[string]interface{}{
			"type":  "gameState",
			"state": g.State,
		})
	}
}

func (g *Game) AddPlayer(p *player.Player) bool {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	if len(g.Players) >= MAX_PLAYERS {
		return false
	}

	// Assign player number
	p.Number = len(g.Players) + 1
	p.GameID = g.ID

	// Add to players map
	g.Players[p.ID] = p

	// Update game state
	g.State.PlayersReady = append(g.State.PlayersReady, false)
	
	return true
}