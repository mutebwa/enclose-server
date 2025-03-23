package handler

import (
	"fmt"
	"log"
	"net/http"

	"enclose/internal/hub"
	"enclose/internal/player"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	EnableCompression: true,
	CheckOrigin:      func(r *http.Request) bool { return true },
}

type WebSocketHandler struct {
	Hub *hub.Hub
}

func (h *WebSocketHandler) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	player := player.New(conn)
	defer h.HandleDisconnect(player)

	h.InitializePlayer(player)
	h.HandleMessages(player)
}

func (h *WebSocketHandler) InitializePlayer(p *player.Player) {
	game := h.Hub.FindOrCreateGame()
	if !game.AddPlayer(p) {
		p.SendJSON(map[string]interface{}{
			"type":    "error",
			"message": "Game is full",
		})
		return
	}
	p.SendJSON(map[string]interface{}{
		"type": "connection",
		"gameId": game.ID,
		"playerId": p.ID,
		"playerNumber": p.Number,
	})
}

func (h *WebSocketHandler) HandleMessages(p *player.Player) {
	for {
		var msg map[string]interface{}
		if err := p.Conn.ReadJSON(&msg); err != nil {
			break
		}
		
		p.UpdateActivity()
		h.ProcessMessage(p, msg)
	}
}

func (h *WebSocketHandler) ProcessMessage(p *player.Player, msg map[string]interface{}) {
	switch msg["type"] {
	case "move":
		h.HandleMove(p, msg)
	case "ready":
		h.HandleReady(p)
	case "reset":
		h.HandleReset(p)
	}
}


func (h *WebSocketHandler) HandleReady(p *player.Player) {
	game := h.Hub.GetGame(p.GameID)
	if game == nil {
		return
	}

	game.Mu.Lock()
	defer game.Mu.Unlock()

	// Mark player ready
	game.State.PlayersReady[p.Number-1] = true

	// Check if all players are ready
	if game.AllPlayersReady() {
		game.State.Message = "Game starting!"
		game.Start()
	}
	
	game.BroadcastState()
}

func (h *WebSocketHandler) HandleReset(p *player.Player) {
	game := h.Hub.GetGame(p.GameID)
	if game == nil {
		return
	}

	game.Mu.Lock()
	defer game.Mu.Unlock()
	
	game.Reset()
	game.BroadcastState()
}

func (h *WebSocketHandler) HandleDisconnect(p *player.Player) {
	game := h.Hub.GetGame(p.GameID)
	if game == nil {
		return
	}

	game.Mu.Lock()
	defer game.Mu.Unlock()
	
	// Remove player from game
	delete(game.Players, p.ID)
	
	// Cleanup empty games
	if len(game.Players) == 0 {
		h.Hub.RemoveGame(game.ID)
	} else {
		// Notify remaining players
		game.State.Message = fmt.Sprintf("Player %d disconnected", p.Number)
		game.BroadcastState()
	}
}