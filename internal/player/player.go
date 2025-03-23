package player

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)


type Player struct {
	ID       string
	Conn     *websocket.Conn
	Number   int
	GameID   string  // Add this field
	LastSeen time.Time
	mu       sync.Mutex
}

func New(conn *websocket.Conn) *Player {
	return &Player{
		ID:       "player-" + time.Now().Format("20060102150405"),
		Conn:     conn,
		LastSeen: time.Now(),
	}
}

func (p *Player) UpdateActivity() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.LastSeen = time.Now()
}

func (p *Player) SendJSON(v interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Conn.WriteJSON(v)
}