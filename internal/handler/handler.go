package handler

import (
	"encoding/json"
	"log"
	"Game_KataBaku/internal/game"
	"Game_KataBaku/internal/hub"
	"Game_KataBaku/internal/types"
	"github.com/gorilla/websocket"
)

type Handler struct {
	hub *hub.Hub
}

func NewHandler(h *hub.Hub) *Handler {
	return &Handler{hub: h}
}

func (h *Handler) HandleMessage(conn *websocket.Conn, msg types.WsMessage) {
	switch msg.Type {
	case "create_room":
		h.handleCreateRoom(conn, msg.Payload)
	case "join_room":
		h.handleJoinRoom(conn, msg.Payload)
	case "start_game":
		h.handleStartGame(conn)
	case "play_cards":
		h.handlePlayCards(conn, msg.Payload)
	case "draw_card":
		h.handleDrawCard(conn)
	case "change_main_card":
		h.handleChangeMainCard(conn)
	default:
		h.sendError(conn, "unknown message type: "+msg.Type)
	}
}

func (h *Handler) handleCreateRoom(conn *websocket.Conn, payload interface{}) {
	data, _ := json.Marshal(payload)
	var req types.CreateRoomMsg
	json.Unmarshal(data, &req)

	if req.Username == "" {
		h.sendError(conn, "username required")
		return
	}

	client := h.hub.GetClient(conn)
	if client != nil {
		client.Username = req.Username
	}

	code := h.hub.CreateRoom(client)

	log.Printf("Room created: %s by %s", code, req.Username)

	conn.WriteJSON(types.WsMessage{
		Type:    "room_created",
		Payload: types.RoomCreatedPayload{
			RoomCode: code,
			PlayerID: client.ID,
		},
	})

	// Also send connection_info with player ID
	conn.WriteJSON(types.WsMessage{
		Type:    "connection_info",
		Payload: map[string]string{"playerId": client.ID},
	})

	h.broadcastRoomState(code)
}

func (h *Handler) handleJoinRoom(conn *websocket.Conn, payload interface{}) {
	// Get client first
	client := h.hub.GetClient(conn)
	if client == nil {
		h.sendError(conn, "client not found")
		return
	}

	data, _ := json.Marshal(payload)
	var req types.JoinRoomMsg
	json.Unmarshal(data, &req)

	if req.Username == "" {
		h.sendError(conn, "username required")
		return
	}
	if req.RoomCode == "" {
		h.sendError(conn, "room code required")
		return
	}

	client.Username = req.Username

	// Send connection_info FIRST before join
	conn.WriteJSON(types.WsMessage{
		Type:    "connection_info",
		Payload: map[string]string{"playerId": client.ID},
	})

	_, err := h.hub.JoinRoom(req.RoomCode, conn)
	if err != nil {
		h.sendError(conn, err.Error())
		return
	}

	log.Printf("Player %s joined room %s", req.Username, req.RoomCode)

	h.broadcastRoomState(req.RoomCode)
}

func (h *Handler) handleStartGame(conn *websocket.Conn) {
	client := h.hub.GetClient(conn)
	if client == nil || client.RoomCode == "" {
		h.sendError(conn, "not in a room")
		return
	}

	room := h.hub.GetRoom(client.RoomCode)
	if room == nil {
		h.sendError(conn, "room not found")
		return
	}

	if room.HostID != client.ID {
		h.sendError(conn, "only host can start game")
		return
	}

	game := h.hub.GetGame(client.RoomCode)
	if game == nil {
		h.sendError(conn, "game not found")
		return
	}

	log.Printf("DEBUG: game.Players count = %d, game.Status = %s", len(game.Players), game.Status)

	if len(game.Players) < 1 {
		h.sendError(conn, "need at least 1 player")
		return
	}

	err := game.Start()
	if err != nil {
		log.Printf("DEBUG: game.Start() error = %v", err)
		h.sendError(conn, err.Error())
		return
	}

	log.Printf("DEBUG: game started successfully!")

	room.Status = "playing"

	log.Printf("Game started in room %s", client.RoomCode)

	h.broadcastGameState(client.RoomCode)
}

func (h *Handler) handlePlayCards(conn *websocket.Conn, payload interface{}) {
	client := h.hub.GetClient(conn)
	if client == nil || client.RoomCode == "" {
		h.sendError(conn, "not in a room")
		return
	}

	data, _ := json.Marshal(payload)
	var req struct {
		Cards    []string `json:"cards"`
		Position string   `json:"position"`
	}
	json.Unmarshal(data, &req)

	if len(req.Cards) == 0 {
		h.sendError(conn, "pilihMinimal satu kartu")
		return
	}

	if req.Position != "prefix" && req.Position != "suffix" {
		req.Position = "suffix"
	}

	game := h.hub.GetGame(client.RoomCode)
	if game == nil {
		h.sendError(conn, "game not found")
		return
	}

	result := game.PlayCards(client.ID, req.Cards, req.Position)

	if !result.Valid {
		h.sendError(conn, result.Message)
		h.sendPlayResult(conn, result)
		return
	}

	// Broadcast correct answer to ALL clients in room
	h.broadcastCorrectAnswer(client.RoomCode, client.ID, result.Word)

	winnerID, isWinner := game.CheckWinner()
	if isWinner {
		h.broadcastGameOver(client.RoomCode, winnerID)
		return
	}

	h.broadcastGameState(client.RoomCode)
	h.sendPlayResult(conn, result)
}

func (h *Handler) handleDrawCard(conn *websocket.Conn) {
	client := h.hub.GetClient(conn)
	if client == nil || client.RoomCode == "" {
		h.sendError(conn, "not in a room")
		return
	}

	game := h.hub.GetGame(client.RoomCode)
	if game == nil {
		h.sendError(conn, "game not found")
		return
	}

	_, err := game.DrawCard(client.ID)
	if err != nil {
		h.sendError(conn, err.Error())
		return
	}

	h.broadcastGameState(client.RoomCode)
}

func (h *Handler) handleChangeMainCard(conn *websocket.Conn) {
	client := h.hub.GetClient(conn)
	if client == nil || client.RoomCode == "" {
		h.sendError(conn, "not in a room")
		return
	}

	game := h.hub.GetGame(client.RoomCode)
	if game == nil {
		h.sendError(conn, "game not found")
		return
	}

	oldCard, err := game.ChangeMainCard()
	if err != nil {
		h.sendError(conn, err.Error())
		return
	}

	log.Printf("Player %s changed main card from %s to %s", client.Username, oldCard, game.MainCard)

	h.broadcastGameState(client.RoomCode)
}

func (h *Handler) sendError(conn *websocket.Conn, message string) {
	conn.WriteJSON(types.WsMessage{
		Type:    "error",
		Payload: types.ErrorPayload{Message: message},
	})
}

func (h *Handler) sendPlayResult(conn *websocket.Conn, result *game.PlayResult) {
	type playResultPayload struct {
		Valid     bool   `json:"valid"`
		Word     string `json:"word,omitempty"`
		Message  string `json:"message"`
		NewMainCard string `json:"newMainCard,omitempty"`
	}

	conn.WriteJSON(types.WsMessage{
		Type:    "play_result",
		Payload: playResultPayload{
			Valid:      result.Valid,
			Word:      result.Word,
			Message:   result.Message,
			NewMainCard: result.NewMainCard,
		},
	})
}

func (h *Handler) broadcastRoomState(roomCode string) {
	room := h.hub.GetRoom(roomCode)
	if room == nil {
		return
	}

	players := getPlayersFromRoom(room)

	conns := make([]*websocket.Conn, 0, len(room.Clients))
	for c := range room.Clients {
		conns = append(conns, c)
	}

	state := types.WsMessage{
		Type:    "room_state",
		Payload: types.RoomState{
			RoomCode: roomCode,
			Status:   room.Status,
			Players:  players,
		},
	}

	for _, c := range conns {
		c.WriteJSON(state)
	}
}

func (h *Handler) broadcastGameState(roomCode string) {
	game := h.hub.GetGame(roomCode)
	if game == nil {
		return
	}

	state := game.GetState()
	log.Printf("DEBUG: Broadcasting game_state - mainCard=%s, players=%d", state.MainCard, len(state.Players))
	for _, p := range state.Players {
		log.Printf("DEBUG: Player %s has cards: %v", p.Username, p.Cards)
	}

	room := h.hub.GetRoom(roomCode)
	if room == nil {
		return
	}

	conns := make([]*websocket.Conn, 0, len(room.Clients))
	for c := range room.Clients {
		conns = append(conns, c)
	}

	msg := types.WsMessage{
		Type:    "game_state",
		Payload: state,
	}

	for _, c := range conns {
		c.WriteJSON(msg)
	}
}

func (h *Handler) broadcastGameOver(roomCode string, winnerID string) {
	room := h.hub.GetRoom(roomCode)
	game := h.hub.GetGame(roomCode)
	if room == nil || game == nil {
		return
	}

	winnerName := ""
	for _, p := range game.Players {
		if p.ID == winnerID {
			winnerName = p.Username
			break
		}
	}

	conns := make([]*websocket.Conn, 0, len(room.Clients))
	for c := range room.Clients {
		conns = append(conns, c)
	}

	type gameOverPayload struct {
		WinnerID   string `json:"winnerId"`
		WinnerName string `json:"winnerName"`
		MainCard  string `json:"mainCard"`
	}

	msg := types.WsMessage{
		Type:    "game_over",
		Payload: gameOverPayload{
			WinnerID:   winnerID,
			WinnerName: winnerName,
			MainCard:  game.MainCard,
		},
	}

	for _, c := range conns {
		c.WriteJSON(msg)
	}
}

func getPlayersFromRoom(room *types.Room) []types.Player {
	players := make([]types.Player, 0, len(room.Clients))
	for _, client := range room.Clients {
		players = append(players, types.Player{
			ID:       client.ID,
			Username: client.Username,
			IsHost:   client.IsHost,
		})
	}
	return players
}

func (h *Handler) broadcastCorrectAnswer(roomCode string, playerID string, word string) {
	room := h.hub.GetRoom(roomCode)
	game := h.hub.GetGame(roomCode)
	if room == nil || game == nil {
		return
	}

	playerName := ""
	for _, p := range game.Players {
		if p.ID == playerID {
			playerName = p.Username
			break
		}
	}

	conns := make([]*websocket.Conn, 0, len(room.Clients))
	for c := range room.Clients {
		conns = append(conns, c)
	}

	type correctAnswerPayload struct {
		PlayerName string `json:"playerName"`
		Word      string `json:"word"`
	}

	msg := types.WsMessage{
		Type:    "correct_answer",
		Payload: correctAnswerPayload{
			PlayerName: playerName,
			Word:      word,
		},
	}

	for _, c := range conns {
		c.WriteJSON(msg)
	}
}