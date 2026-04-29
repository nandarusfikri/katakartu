package types

import (
	"github.com/gorilla/websocket"
)

type Client struct {
	ID       string
	Username string
	Conn     *websocket.Conn
	RoomCode string
	IsHost   bool
}

type Room struct {
	Code    string                      `json:"code"`
	HostID  string                      `json:"hostId"`
	Clients map[*websocket.Conn]*Client `json:"-"`
	Status  string                      `json:"status"`
	Players []Player                    `json:"players"`
}

type Player struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Cards    []string `json:"cards,omitempty"`
	Score    int      `json:"score"`
	IsHost   bool     `json:"isHost"`
}

type PlayerState struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	IsHost   bool     `json:"isHost"`
	Cards    []string `json:"cards"`
	Score    int      `json:"score"`
}

type RoomState struct {
	RoomCode  string   `json:"roomCode"`
	Status    string   `json:"status"`
	Players   []Player `json:"players"`
	MainCard  string   `json:"mainCard"`
	Deck      []string `json:"deck"`
	Timestamp int64    `json:"timestamp"`
}

type GameState struct {
	RoomCode    string        `json:"roomCode"`
	Status      string        `json:"status"`
	Players     []PlayerState `json:"players"`
	MainCard    string        `json:"mainCard"`
	Leaderboard []PlayerState `json:"leaderboard"`
	Timer       int           `json:"timer"`
	Timestamp   int64         `json:"timestamp"`
}

type WsMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type CreateRoomMsg struct {
	Username string `json:"username"`
	Duration int    `json:"duration"`
}

type JoinRoomMsg struct {
	RoomCode string `json:"roomCode"`
	Username string `json:"username"`
}

type StartGameMsg struct {
}

type RoomCreatedPayload struct {
	RoomCode string `json:"roomCode"`
	PlayerID string `json:"playerId"`
}

type JoinRoomPayload struct {
	RoomCode string `json:"roomCode"`
	PlayerID string `json:"playerId"`
}

type PlayerJoinedPayload struct {
	Players []Player `json:"players"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

type VoteRequestPayload struct {
	InitiatorName string `json:"initiatorName"`
	NewMainCard   string `json:"newMainCard"`
	OldMainCard   string `json:"oldMainCard"`
	TotalPlayers  int    `json:"totalPlayers"`
	SecondsLeft   int    `json:"secondsLeft"`
}

type VoteResultPayload struct {
	Approved int    `json:"approved"`
	Rejected int    `json:"rejected"`
	MainCard string `json:"mainCard"`
	Success  bool   `json:"success"`
	Message  string `json:"message"`
}
