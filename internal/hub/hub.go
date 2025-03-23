package hub

import (
	"sync"
	"time"
	
	"enclose/internal/game"
)

type Hub struct {
    games      map[string]*game.Game
    mu         sync.RWMutex
    maxGames   int
    gameCount  int
}

func NewHub(maxGames int) *Hub {
    return &Hub{
        games:    make(map[string]*game.Game),
        maxGames: maxGames,
    }
}

func (h *Hub) MaintainGames(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		h.CleanupExpiredGames()
	}
}

func (h *Hub) CleanupExpiredGames() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, g := range h.games {
		if g.IsExpired() || g.IsEmpty() {
			delete(h.games, id)
		}
	}
}

func (h *Hub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	for _, g := range h.games {
		g.Stop()
	}
	h.games = make(map[string]*game.Game)
}

func (h *Hub) FindOrCreateGame() *game.Game {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.gameCount >= h.maxGames {
        return nil // Or handle queueing
    }

	// Remove expired games first
	for id, g := range h.games {
		if g.IsExpired() {
			delete(h.games, id)
		}
	}

	// Find available game
	for _, g := range h.games {
		if !g.IsFull() {
			return g
		}
	}

	// Create new game
	newGame := game.NewGame()
	h.games[newGame.ID] = newGame
	return newGame
}
func (h *Hub) GetGame(gameID string) *game.Game {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.games[gameID]
}

func (h *Hub) RemoveGame(gameID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.games, gameID)
}