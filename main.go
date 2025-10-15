package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func main() {
	router := gin.Default()
	router.StaticFile("/", "index.html")
	router.StaticFile("/msg-css", "message.css")
	router.GET("/ws", serveWs)
	//fmt.Println("No of connections ")
	err := router.Run()
	if err != nil {
		log.Fatalf("Unable to start server. Error %v", err)
	}
	log.Println("Server started successfully.")
}

var userId string

func serveWs(c *gin.Context) {

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	//fmt.Println("c.Request --------------------- ", c.Query("coupe_id"))
	if err != nil {
		log.Printf("Error in upgrading web socket. Error: %v", err)
		return
	}
	userId = c.Query("coupe_id")
	go handleClient(conn)
}

var clients = make(map[*websocket.Conn]struct{})
var client = make(map[string][]*websocket.Conn)

type Message struct {
	From     string `json:"from"`
	Message  string `json:"message"`
	SentTo   string `json:"to"`
	NoOfConn int    `json:"users"`
}

func handleClient(c *websocket.Conn) {
	defer func() {
		delete(clients, c)
		log.Println("Closing Websocket")
		c.Close()
	}()
	clients[c] = struct{}{}
	client[userId] = append(client[userId], c)
	for {
		var msg Message
		err := c.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error in reading json message. Error : %v", err)
			return
		}
		fmt.Println("msg", msg)
		// process the message
		broadcast(msg)
	}
}

func broadcast(msg Message) {
	msg.NoOfConn = len(client[msg.SentTo])

	for _, conn := range client[msg.SentTo] {
		conn.WriteJSON(msg)
	}
}
