package handler

import (
	"enclose/internal/player"
	g "enclose/internal/game"
)

func (h *WebSocketHandler) HandleMove(p *player.Player, msg map[string]interface{}) {
	x, y, valid := parseCoordinates(msg)
	if !valid {
		return
	}

	game := h.Hub.GetGame(p.GameID)
	if game == nil {
		return
	}

	game.Mu.Lock()
	defer game.Mu.Unlock()

	if validMove(game.State, x, y, p.Number) {
		g.ProcessMove(game, x, y, p.Number)
		game.BroadcastState()
	}
}

func parseCoordinates(msg map[string]interface{}) (int, int, bool) {
	x, ok1 := msg["x"].(float64)
	y, ok2 := msg["y"].(float64)
	return int(x), int(y), ok1 && ok2
}

func validMove(state g.State, x, y, player int) bool {
	return x >= 0 && x < len(state.Grid) &&
		y >= 0 && y < len(state.Grid) &&
		state.Grid[y][x].Owner == 0 &&
		state.CurrentTurn == player
}