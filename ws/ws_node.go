package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"server_monitor/model"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var (
	UpgraderNode = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	ClientsNode = make(map[string]*websocket.Conn)
	MutexNode   = sync.Mutex{}
)

func HandleWebSocketNode(w http.ResponseWriter, r *http.Request, clientID string) {
	connNode, err := UpgraderNode.Upgrade(w, r, nil)
	if err != nil {
		logrus.Error(err)
		log.Println(err)
		return
	}

	MutexNode.Lock()
	ClientsNode[clientID] = connNode
	MutexNode.Unlock()

	go HandleMessagesNode(clientID, connNode)
}

func HandleMessagesNode(clientID string, connNode *websocket.Conn) {
	defer func() {
		MutexNode.Lock()
		delete(ClientsNode, clientID)
		MutexNode.Unlock()

		connNode.Close()
	}()

	for {
		messageType, p, err := connNode.ReadMessage()
		if err != nil {
			logrus.Error(err)
			log.Println(err)
			return
		}

		// log.Printf("Received message from %s", p)

		// Handle the message (you can implement your own message format)
		HandleMessageNode(messageType, p)
	}
}

func HandleMessageNode(messageType int, message []byte) {
	if len(message) == 0 {
		return
	}
	// fmt.Println("stRmessage")
	// fmt.Println("stRmessage")
	// fmt.Println("stRmessage")
	// fmt.Println(string(message))
	// fmt.Println("stRmessage")
	// fmt.Println("stRmessage")
	messageStr := string(message)
	if strings.HasPrefix(messageStr, "server:") {
		go BroadcastMessage(1, messageStr)
		messageStr = strings.TrimPrefix(messageStr, "server:")
		var mapID = make(map[string]bool)
		for data := range strings.SplitSeq(messageStr, ";;") {
			parts := strings.SplitN(data, "::", 2)
			if len(parts) != 2 {
				continue
			}
			value := strings.TrimSpace(parts[1])
			keys := strings.SplitN(strings.TrimSpace(parts[0]), "-", 2)
			if len(keys) != 2 {
				continue
			}
			field := strings.TrimSpace(keys[0])
			id := strings.ReplaceAll(keys[1], "_", ".")
			err := model.UpdateServerCache(id, map[string]interface{}{field: value})
			if err != nil {
				logrus.Errorf("Failed to update server cache for %s: %v", id, err)
			}
			mapID[id] = true
		}
		go func() {
			for id := range mapID {
				if server, ok := model.ServerCache[id]; ok && server != nil {
					server.LastCheckTime = time.Now()
				}
			}
			mapID = nil
		}()
		return
	} else if strings.HasPrefix(messageStr, "server_services:") {
		messageStr = strings.TrimPrefix(messageStr, "server_services:")
		var serverData map[string][]*model.ServerService
		err := json.Unmarshal([]byte(messageStr), &serverData)
		if err != nil {
			logrus.Errorf("Failed to unmarshal server services message: %v", err)
			return
		}
		model.MuServerServices.Lock()
		maps.Copy(model.ServerServices, serverData)
		model.MuServerServices.Unlock()
		if result, err := json.Marshal(model.ServerServices); err == nil {
			go func(b []byte) {
				cacheDir := "./.cache"
				filePath := filepath.Join(cacheDir, "server_services.json")

				// Cek folder dan buat kalau belum ada
				if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
					if err := os.MkdirAll(cacheDir, 0755); err != nil {
						log.Fatalf("❌ Gagal membuat folder cache: %v", err)
					}
				}

				// Simpan file
				if err := os.WriteFile(filePath, b, 0777); err != nil {
					log.Fatalf("❌ Gagal menyimpan file: %v", err)
				}

				fmt.Printf("💾 Berhasil menyimpan data ke %s\n", filePath)
			}(result)
		}
		return
	} else if strings.TrimSpace(messageStr)[0] != '{' {
		return
	}
	// fmt.Println("messageX")
	// fmt.Println("messageX")
	// fmt.Println("messageX")
	// fmt.Println(string(message))
	// fmt.Println("messageX")
	// fmt.Println("message")
	// fmt.Println("messageX")
	// fmt.Println("message")
	// fmt.Println("messageX")

	messageStr = strings.TrimPrefix(messageStr, "server:")

	var serverData map[string]map[string]interface{}
	err := json.Unmarshal([]byte(messageStr), &serverData)
	if err != nil {
		logrus.Error(messageStr)
		logrus.Errorf("Failed to unmarshal message: %v", err)
		return
	}
	for serverID, data := range serverData {
		if net.ParseIP(serverID) == nil {
			continue
		}

		err := model.UpdateServerCache(serverID, data)
		if err != nil {
			logrus.Errorf("Failed to update server cache for %s: %v", serverID, err)
		}
		if server, ok := model.ServerCache[serverID]; ok && server != nil {
			server.LastCheckTime = time.Now()
		}
	}
}

// SendMessageToRecipientNode(1, "the message", "email") //1 is text message, 2 is binary
func SendMessageToRecipientNode(messageType int, message, recipientID string) {
	MutexNode.Lock()
	defer MutexNode.Unlock()

	if clientConnNode, ok := ClientsNode[recipientID]; ok {
		if clientConnNode != nil {
			err := clientConnNode.WriteMessage(messageType, []byte(message))
			if err != nil {
				logrus.Error(err)
				log.Println(err)
			}
		}
	}
}

// BroadcastMessage sends a message to all connNodeected clients.
// Text Message: 1 (or websocket.TextMessage)
// Binary Message: 2 (or websocket.BinaryMessage)
func BroadcastMessageNode(messageType int, message string) {
	MutexNode.Lock()
	defer MutexNode.Unlock()

	for clientID, clientConnNode := range ClientsNode {
		if clientConnNode != nil && !strings.HasPrefix(clientID, "broker_mqtt") {
			err := clientConnNode.WriteMessage(messageType, []byte(message))
			if err != nil {
				logrus.Error(err)
				log.Printf("Error sending message to %s: %v", clientID, err)
			}
		}
	}
}

func CloseWebsocketConnectionNode(clientID string) {
	MutexNode.Lock()
	if clientConnNode, ok := ClientsNode[clientID]; ok {
		if clientConnNode != nil {
			clientConnNode.Close()
		}
		delete(ClientsNode, clientID)
	}
	MutexNode.Unlock()
}
