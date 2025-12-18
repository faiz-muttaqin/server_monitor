package ws

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var (
	Upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	Clients = make(map[string]*websocket.Conn)
	Mutex   = sync.Mutex{}
)

func HandleWebSocket(w http.ResponseWriter, r *http.Request, clientID string) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Error(err)
		log.Println(err)
		return
	}

	Mutex.Lock()
	Clients[clientID] = conn
	Mutex.Unlock()

	go HandleMessages(clientID, conn)
}

func HandleMessages(clientID string, conn *websocket.Conn) {
	defer func() {
		Mutex.Lock()
		delete(Clients, clientID)
		Mutex.Unlock()

		conn.Close()
	}()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			logrus.Error(err)
			log.Println(err)
			return
		}

		// log.Printf("Received message from %s", p)

		// Handle the message (you can implement your own message format)
		HandleMessage(messageType, p)
	}
}

func HandleMessage(messageType int, message []byte) {
	// Example: Assume messages have the format "recipientID:message"
	parts := strings.SplitN(string(message), ":", 2)
	if len(parts) != 2 {
		log.Println("Invalid message format:", string(message))
		return
	}

	recipientID := parts[0]
	actualMessage := parts[1]

	// Broadcast the message to the intended recipient
	SendMessageToRecipient(messageType, actualMessage, recipientID)
}

// SendMessageToRecipient(1, "the message", "email") //1 is text message, 2 is binary
func SendMessageToRecipient(messageType int, message, recipientID string) {
	Mutex.Lock()
	defer Mutex.Unlock()

	if clientConn, ok := Clients[recipientID]; ok {
		if clientConn != nil {
			err := clientConn.WriteMessage(messageType, []byte(message))
			if err != nil {
				logrus.Error(err)
				log.Println(err)
			}
		}
	}
}

// BroadcastMessage sends a message to all connected clients.
// Text Message: 1 (or websocket.TextMessage)
// Binary Message: 2 (or websocket.BinaryMessage)
func BroadcastMessage(messageType int, message string) {
	Mutex.Lock()
	defer Mutex.Unlock()

	for clientID, clientConn := range Clients {
		if clientConn != nil && !strings.HasPrefix(clientID, "broker_mqtt") {
			err := clientConn.WriteMessage(messageType, []byte(message))
			if err != nil {
				logrus.Error(err)
				log.Printf("Error sending message to %s: %v", clientID, err)
			}
		}
	}
}

// BroadcastMessage sends a message to all connected clients.
// Text Message: 1 (or websocket.TextMessage)
// Binary Message: 2 (or websocket.BinaryMessage)
func BroadcastMessageToMqttBroker(messageType int, message string) {
	Mutex.Lock()
	defer Mutex.Unlock()

	for clientID, clientConn := range Clients {
		if strings.HasPrefix(clientID, "broker_mqtt") && clientConn != nil {
			// fmt.Println("clientID")
			// fmt.Println(clientID)
			// fmt.Println("message")
			// fmt.Println(message)
			logrus.Infof("Sending message to MQTT broker client %s: %s", clientID, message)
			err := clientConn.WriteMessage(messageType, []byte(message))
			if err != nil {
				logrus.Printf("Error sending MQTT message to %s: %v", clientID, err)
			}
		}
	}
}

func CloseWebsocketConnection(clientID string) {
	Mutex.Lock()
	if clientConn, ok := Clients[clientID]; ok {
		if clientConn != nil {
			clientConn.Close()
		}
		delete(Clients, clientID)
	}
	Mutex.Unlock()
}
