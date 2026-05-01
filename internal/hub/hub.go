package hub

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"Game_KataBaku/internal/game"
	"Game_KataBaku/internal/types"
	"github.com/gorilla/websocket"
)

type Hub struct {
	rooms   map[string]*types.Room
	clients map[*websocket.Conn]*types.Client
	games   map[string]*game.Game
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		rooms:   make(map[string]*types.Room),
		clients: make(map[*websocket.Conn]*types.Client),
		games:   make(map[string]*game.Game),
	}
}

func (h *Hub) RegisterClient(conn *websocket.Conn, clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[conn] = &types.Client{
		ID:       clientID,
		Username: "",
		Conn:     conn,
		RoomCode: "",
		IsHost:   false,
	}
}

func (h *Hub) GetGame(code string) *game.Game {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.games[code]
}

func (h *Hub) RemoveClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, exists := h.clients[conn]
	if !exists {
		return
	}

	if client.RoomCode != "" {
		room, exists := h.rooms[client.RoomCode]
		if exists {
			delete(room.Clients, conn)
			if len(room.Clients) == 0 {
				delete(h.rooms, client.RoomCode)
				delete(h.games, client.RoomCode)
			}
		}
	}

	delete(h.clients, conn)
}

func (h *Hub) CreateRoom(host *types.Client) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	code := generateRoomCode()
	room := &types.Room{
		Code:    code,
		HostID:  host.ID,
		Clients: make(map[*websocket.Conn]*types.Client),
		Status:  "waiting",
	}
	room.Clients[host.Conn] = host
	host.RoomCode = code
	host.IsHost = true
	h.rooms[code] = room

	g := game.NewGame(code)
	h.games[code] = g
	g.AddPlayer(host.ID, host.Username, true)

	return code
}

func (h *Hub) JoinRoom(code string, conn *websocket.Conn) (*types.Room, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[code]
	if !exists {
		return nil, fmt.Errorf("room not found")
	}

	if len(room.Clients) >= 6 {
		return nil, fmt.Errorf("room full (max 6 players)")
	}

	client, ok := h.clients[conn]
	if !ok {
		return nil, fmt.Errorf("client not registered")
	}

	room.Clients[conn] = client
	client.RoomCode = code

	if g, ok := h.games[code]; ok {
		g.AddPlayer(client.ID, client.Username, false)
	}

	return room, nil
}

func (h *Hub) GetRoom(code string) *types.Room {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.rooms[code]
}

func (h *Hub) GetClient(conn *websocket.Conn) *types.Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.clients[conn]
}

func (h *Hub) BroadcastToRoom(roomCode string, msg types.WsMessage) {
	h.mu.Lock()
	room := h.rooms[roomCode]
	if room == nil {
		h.mu.Unlock()
		return
	}

	var failedConns []*websocket.Conn
	for conn := range room.Clients {
		err := conn.WriteJSON(msg)
		if err != nil {
			failedConns = append(failedConns, conn)
		}
	}

	for _, conn := range failedConns {
		conn.Close()
		delete(room.Clients, conn)
		if client, exists := h.clients[conn]; exists {
			delete(h.clients, conn)
			if g, ok := h.games[roomCode]; ok {
				g.RemovePlayer(client.ID)
			}
		}
	}

	if len(room.Clients) == 0 {
		delete(h.rooms, roomCode)
		delete(h.games, roomCode)
	}

	h.mu.Unlock()
}

func (h *Hub) SendToClient(conn *websocket.Conn, msg types.WsMessage) {
	conn.WriteJSON(msg)
}

func generateRoomCode() string {
	rand.Seed(time.Now().UnixNano())
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	code := make([]byte, 4)
	for i := range code {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return string(code)
}
