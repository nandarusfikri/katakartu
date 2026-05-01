package main

import (
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	gameHandler "Game_KataBaku/internal/handler"
	"Game_KataBaku/internal/hub"
	"Game_KataBaku/internal/types"
	"github.com/gorilla/websocket"
)

const (
	HOST = ":8080"
	PORT = "8080"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var gameHub *hub.Hub
var handler *gameHandler.Handler

func main() {
	baseDir := getBaseDir()
	webDir := filepath.Join(baseDir, "web")

	gameHub = hub.NewHub()
	handler = gameHandler.NewHandler(gameHub)

	http.Handle("/", http.FileServer(http.Dir(webDir)))
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/words", handleWords)

	port := os.Getenv("PORT")
	if port == "" {
		port = PORT
	}
	addr := ":" + port

	log.Printf("Server running at http://localhost%s", addr)
	log.Printf("Serving static files from %s", webDir)
	log.Printf("Word list available at http://localhost%s/words", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	clientID := generateClientID()
	gameHub.RegisterClient(conn, clientID)

	log.Printf("Client connected: %s", clientID)
	go handleClient(conn)
}

func handleClient(conn *websocket.Conn) {
	defer func() {
		gameHub.RemoveClient(conn)
		conn.Close()
	}()

	for {
		var msg types.WsMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if !websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break
			}
			log.Printf("Read error: %v", err)
			break
		}

		log.Printf("Received: %+v", msg)
		handler.HandleMessage(conn, msg)
	}
}

func handleWords(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	baseDir := getBaseDir()
	dataFile := filepath.Join(baseDir, "data", "kata.txt")
	data, err := os.ReadFile(dataFile)
	if err != nil {
		http.Error(w, "File not found", 404)
		return
	}

	w.Write(data)
}

func getBaseDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

func generateClientID() string {
	rand.Seed(time.Now().UnixNano())
	return randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
