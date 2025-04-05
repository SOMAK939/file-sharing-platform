package handlers

import (
    "fmt"
    "net/http"
    "sync"

    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]bool)
var mu sync.Mutex

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(" WebSocket upgrade failed:", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	fmt.Println(" WebSocket connection established")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(" WebSocket error:", err)
			break
		}
		fmt.Printf(" Received message: %s\n", msg)
	}
}

func NotifyUploadComplete(filename, userID string) {
    mu.Lock()
    defer mu.Unlock()

    message := fmt.Sprintf("File %s has been uploaded successfully", filename)
    for client := range clients {
        err := client.WriteMessage(websocket.TextMessage, []byte(message))
        if err != nil {
            client.Close()
            delete(clients, client)
        }
    }
}
