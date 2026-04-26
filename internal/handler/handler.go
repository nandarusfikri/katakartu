package handler

import (
	"Game_KataBaku/internal/game"
	"Game_KataBaku/internal/hub"
	"Game_KataBaku/internal/types"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"time"
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
	case "request_change_main":
		h.handleRequestChangeMain(conn, msg.Payload)
	case "vote_response":
		h.handleVoteResponse(conn, msg.Payload)
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
		Type: "room_created",
		Payload: types.RoomCreatedPayload{
			RoomCode: code,
			PlayerID: client.ID,
		},
	})

	conn.WriteJSON(types.WsMessage{
		Type:    "connection_info",
		Payload: map[string]string{"playerId": client.ID},
	})

	h.broadcastRoomState(code)
}

func (h *Handler) handleJoinRoom(conn *websocket.Conn, payload interface{}) {
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

	g := h.hub.GetGame(client.RoomCode)
	if g == nil {
		h.sendError(conn, "game not found")
		return
	}

	if len(g.Players) < 1 {
		h.sendError(conn, "need at least 1 player")
		return
	}

	err := g.Start()
	if err != nil {
		h.sendError(conn, err.Error())
		return
	}

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
		PrefixCards []string `json:"prefixCards"`
		SuffixCards []string `json:"suffixCards"`
	}
	json.Unmarshal(data, &req)

	game := h.hub.GetGame(client.RoomCode)
	if game == nil {
		h.sendError(conn, "game not found")
		return
	}

	result := game.PlayCards(client.ID, req.PrefixCards, req.SuffixCards)

	if !result.Valid {
		h.sendError(conn, result.Message)
		h.broadcastPlayResult(client.RoomCode, client.ID, result)
		h.broadcastGameState(client.RoomCode)
		return
	}

	h.broadcastPlayResult(client.RoomCode, client.ID, result)

	winnerID, isWinner := game.CheckWinner()
	if isWinner {
		h.broadcastGameOver(client.RoomCode, winnerID)
		return
	}

	h.broadcastGameState(client.RoomCode)
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

func (h *Handler) handleRequestChangeMain(conn *websocket.Conn, payload interface{}) {
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

	if game.PendingVote != nil {
		h.sendError(conn, "sedang ada voting aktif")
		return
	}

	vote := game.CreateVote(client.ID, client.Username)
	if vote == nil {
		h.sendError(conn, "gagal membuat voting")
		return
	}

	h.broadcastVoteRequest(client.RoomCode, vote)

	go h.checkAndExecuteVote(client.RoomCode)
}

func (h *Handler) handleVoteResponse(conn *websocket.Conn, payload interface{}) {
	client := h.hub.GetClient(conn)
	if client == nil {
		return
	}

	data, _ := json.Marshal(payload)
	var req struct {
		Approved bool `json:"approved"`
	}
	json.Unmarshal(data, &req)

	game := h.hub.GetGame(client.RoomCode)
	if game == nil {
		return
	}

	processed, _ := game.ProcessVoteResponse(client.ID, req.Approved)
	if processed {
		h.broadcastVoteProgress(client.RoomCode, game.PendingVote)
	}
}

func (h *Handler) checkAndExecuteVote(roomCode string) {
	time.Sleep(5 * time.Second)

	game := h.hub.GetGame(roomCode)
	if game == nil {
		return
	}

	vote := game.ExecuteVoteIfExpired()
	if vote == nil {
		return
	}

	success := game.ExecuteVote(vote)

	h.broadcastVoteResult(roomCode, vote, success)

	if success {
		h.broadcastGameState(roomCode)
	}

	log.Printf("Vote completed: approve=%d, reject=%d, success=%v",
		len(vote.Approves), len(vote.Rejects), success)
}

func (h *Handler) broadcastVoteRequest(roomCode string, vote *game.VoteSession) {
	room := h.hub.GetRoom(roomCode)
	if room == nil {
		return
	}

	msg := types.WsMessage{
		Type: "vote_request",
		Payload: types.VoteRequestPayload{
			InitiatorName: vote.InitiatorName,
			NewMainCard:   vote.NewMainCard,
			OldMainCard:   vote.OldMainCard,
			TotalPlayers:  vote.TotalPlayers,
			SecondsLeft:   5,
		},
	}

	for conn := range room.Clients {
		conn.WriteJSON(msg)
	}
}

func (h *Handler) broadcastVoteProgress(roomCode string, vote *game.VoteSession) {
	if vote == nil {
		return
	}

	room := h.hub.GetRoom(roomCode)
	if room == nil {
		return
	}

	type VoteProgressPayload struct {
		Approved int `json:"approved"`
		Rejected int `json:"rejected"`
	}

	msg := types.WsMessage{
		Type: "vote_progress",
		Payload: VoteProgressPayload{
			Approved: len(vote.Approves),
			Rejected: len(vote.Rejects),
		},
	}

	for conn := range room.Clients {
		conn.WriteJSON(msg)
	}
}

func (h *Handler) broadcastVoteResult(roomCode string, vote *game.VoteSession, success bool) {
	room := h.hub.GetRoom(roomCode)
	if room == nil {
		return
	}

	message := "Ganti main card DITOLAK"
	if success {
		message = "Ganti main card DISETUJUI"
	}

	msg := types.WsMessage{
		Type: "vote_result",
		Payload: types.VoteResultPayload{
			Approved: len(vote.Approves),
			Rejected: len(vote.Rejects),
			MainCard: vote.NewMainCard,
			Success:  success,
			Message:  message,
		},
	}

	for conn := range room.Clients {
		conn.WriteJSON(msg)
	}
}

func (h *Handler) sendError(conn *websocket.Conn, message string) {
	conn.WriteJSON(types.WsMessage{
		Type:    "error",
		Payload: types.ErrorPayload{Message: message},
	})
}

func (h *Handler) broadcastRoomState(roomCode string) {
	room := h.hub.GetRoom(roomCode)
	if room == nil {
		return
	}

	players := getPlayersFromRoom(room)

	for conn := range room.Clients {
		conn.WriteJSON(types.WsMessage{
			Type: "room_state",
			Payload: types.RoomState{
				RoomCode: roomCode,
				Status:   room.Status,
				Players:  players,
			},
		})
	}
}

func (h *Handler) broadcastGameState(roomCode string) {
	game := h.hub.GetGame(roomCode)
	if game == nil {
		return
	}

	room := h.hub.GetRoom(roomCode)
	if room == nil {
		return
	}

	state := game.GetState()

	msg := types.WsMessage{
		Type:    "game_state",
		Payload: state,
	}

	for conn := range room.Clients {
		conn.WriteJSON(msg)
	}
}

func (h *Handler) broadcastPlayResult(roomCode string, playerID string, result *game.PlayResult) {
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

	type playResultPayload struct {
		Valid       bool   `json:"valid"`
		PlayerName  string `json:"playerName"`
		Word        string `json:"word,omitempty"`
		Message     string `json:"message"`
		NewMainCard string `json:"newMainCard,omitempty"`
	}

	msg := types.WsMessage{
		Type: "play_result",
		Payload: playResultPayload{
			Valid:       result.Valid,
			PlayerName:  playerName,
			Word:        result.Word,
			Message:     result.Message,
			NewMainCard: result.NewMainCard,
		},
	}

	for conn := range room.Clients {
		conn.WriteJSON(msg)
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

	type gameOverPayload struct {
		WinnerID   string `json:"winnerId"`
		WinnerName string `json:"winnerName"`
		MainCard   string `json:"mainCard"`
	}

	msg := types.WsMessage{
		Type: "game_over",
		Payload: gameOverPayload{
			WinnerID:   winnerID,
			WinnerName: winnerName,
			MainCard:   game.MainCard,
		},
	}

	for conn := range room.Clients {
		conn.WriteJSON(msg)
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
