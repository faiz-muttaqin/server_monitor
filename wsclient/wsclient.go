package wsclient

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"server_monitor/model"
	"server_monitor/utils"
	"server_monitor/ws"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var (
	IsWsMasterConnected atomic.Bool
	WsMasterConn        *websocket.Conn
)

func Connect() {

	for {
		if !IsWsMasterConnected.Load() {
			connection, err := connectWithRetry()
			if err != nil {
				log.Printf("Failed to establish connection: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			WsMasterConn = connection
			readMessages(connection)
		}
		time.Sleep(2 * time.Second)
	}
}

func connectWithRetry() (*websocket.Conn, error) {
	serverAddr := os.Getenv("MASTER_HOST")
	if serverAddr == "" {
		return nil, fmt.Errorf("MASTER_HOST environment variable is not set")
	}

	path := "/ws/node"
	urlStr := url.URL{Scheme: "ws", Host: serverAddr, Path: path}
	headers := http.Header{}
	headers.Add("key", utils.SERVER_NODE_AUTH_KEY)
	headers.Add("id", utils.IP)
	conn, _, err := websocket.DefaultDialer.Dial(urlStr.String(), headers)
	if err != nil {
		// return nil, err
		scheme := "ws"

		if strings.HasPrefix(serverAddr, "https://") {
			serverAddr = serverAddr[len("https://"):]
			scheme = "wss"
		} else if strings.HasPrefix(serverAddr, "http://") {
			serverAddr = serverAddr[len("http://"):]
		}

		urlStr = url.URL{Scheme: scheme, Host: serverAddr, Path: path}
		headers.Add("key", utils.SERVER_NODE_AUTH_KEY)
		headers.Add("id", utils.IP)
		conn, _, err = websocket.DefaultDialer.Dial(urlStr.String(), headers)
		if err != nil {
			logrus.Error(err)
			// log.Println()
			// fmt.Println("MQTT BROKER ERROR ", pub.ServiceName, " connected to main Service via ", urlStr.String(), ", Error: ", err)
			// logrus.Error("MQTT BROKER ERROR ", pub.ServiceName, " connected to main Service via ", urlStr.String(), ", Error: ", err)
			// log.Println()
			return nil, err
		}
	}
	IsWsMasterConnected.Store(true)
	go ws.BroadcastMessage(1, "master_connected")
	go func() {
		time.Sleep(15 * time.Second)
		cacheJson, err := json.Marshal(model.ServerCache)
		if err != nil {
			logrus.Errorf("Failed to marshal ServerCache: %v", err)
		} else {
			go SendMessage(conn, string(cacheJson))
		}
	}()
	return conn, nil
}

func readMessages(conn *websocket.Conn) {
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Read error: %v", err)
			}
			conn.Close()
			IsWsMasterConnected.Store(false)
			ws.BroadcastMessage(1, "master_disconnected")
			return
		}
		if messageType == websocket.TextMessage {
			fmt.Println(string(message))
			handleMessage(conn, string(message))
		}
	}
}

// handleMessage processes incoming messages and sends a response
func handleMessage(conn *websocket.Conn, message string) {
	// fmt.Println("Recieved : " + message)
	if message == "ping" {
		err := SendMessage(conn, "pong")
		if err != nil {
			log.Printf("Error sending pong response: %v", err)
		}
		return
	}
	// Attempt to parse the message as JSON
	var msgData map[string]interface{}
	if err := json.Unmarshal([]byte(message), &msgData); err != nil {
		fmt.Printf("Invalid JSON received: %s\n", message)
		return
	}

	// Extract the "types" key if it exists
	if types, ok := msgData["types"]; ok {
		// Add logic to handle specific types
		switch types {
		case "ping":
			err := SendMessage(conn, "pong")
			if err != nil {
				log.Printf("Error sending response: %v", err)
			}

		}
	}
}

// SendMessage sends a message to the WebSocket server with error handling
func SendMessage(conn *websocket.Conn, message string) error {
	err := conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		logrus.Printf("Error sending message: %v", err)
		return err
	}
	// fmt.Printf("Sent: %s\n", message)
	return nil
}
