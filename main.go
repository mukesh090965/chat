package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Message struct {
	From    string `json:"from"`
	Message string `json:"message"`
}

type Client struct {
	conn     *websocket.Conn
	coupeID  string
	sendChan chan Message
}

var (
	upgrader    = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	clients     = make(map[string]map[*Client]bool) // coupeID -> set of clients
	clientsLock = sync.RWMutex{}
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/ws", handleWebSocket)

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	coupeID := r.URL.Query().Get("coupe_id")
	if coupeID == "" {
		http.Error(w, "Missing coupe_id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}

	client := &Client{
		conn:     conn,
		coupeID:  coupeID,
		sendChan: make(chan Message),
	}

	clientsLock.Lock()
	if clients[coupeID] == nil {
		clients[coupeID] = make(map[*Client]bool)
	}
	clients[coupeID][client] = true
	clientsLock.Unlock()

	log.Printf("New client connected for coupe_id: %s\n", coupeID)

	go client.readMessages()
	go client.writeMessages()
}

func (c *Client) readMessages() {
	defer c.disconnect()

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		log.Printf("[%s] From %s: %s\n", c.coupeID, msg.From, msg.Message)
		broadcastMessage(c.coupeID, msg, c)
	}
}

func (c *Client) writeMessages() {
	for msg := range c.sendChan {
		err := c.conn.WriteJSON(msg)
		if err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}

func (c *Client) disconnect() {
	log.Printf("Client disconnected from coupe_id: %s\n", c.coupeID)

	clientsLock.Lock()
	delete(clients[c.coupeID], c)
	if len(clients[c.coupeID]) == 0 {
		delete(clients, c.coupeID)
	}
	clientsLock.Unlock()

	c.conn.Close()
	close(c.sendChan)
}

func broadcastMessage(coupeID string, msg Message, sender *Client) {
	clientsLock.RLock()
	defer clientsLock.RUnlock()

	for client := range clients[coupeID] {
		// Skip sender if desired; for signaling you might want to include sender
		if client != sender {
			select {
			case client.sendChan <- msg:
			default:
				log.Println("Send channel full, skipping client")
			}
		}
	}
}
