package controllers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	goravelhttp "github.com/goravel/framework/contracts/http"
)

// allowedWSOrigins — restrict WebSocket connections to known origins.
// FINDING-002 FIX: Validate origin and authenticate user before upgrading.
var allowedWSOrigins = map[string]bool{
	"https://localhost:5173": true,
	"http://localhost:5173":  true,
	"http://localhost:3000":  true,
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false
		}
		// Allow if explicitly listed
		if allowedWSOrigins[origin] {
			return true
		}
		// Also allow APP_ALLOWED_ORIGIN from env (handled at startup if needed)
		return false
	},
}

var clients = make(map[string]map[string]*websocket.Conn) // user_id -> sub_topic_id -> connection
var clientsMu sync.Mutex

type WebSocketController struct {
}

func NewWebSocketController() *WebSocketController {
	return &WebSocketController{}
}

func (c *WebSocketController) HandleWS(ctx goravelhttp.Context) goravelhttp.Response {
	// FINDING-002 FIX: Authenticate from context (set by Auth middleware), not from query param.
	authUserID := ctx.Value("auth_user_id")
	if authUserID == nil {
		w := ctx.Response().Writer()
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}

	userIDStr := ""
	switch v := authUserID.(type) {
	case uint:
		userIDStr = strings.TrimSpace(fmt.Sprintf("%d", v))
	case string:
		userIDStr = strings.TrimSpace(v)
	default:
		userIDStr = fmt.Sprintf("%v", v)
	}
	if userIDStr == "" {
		w := ctx.Response().Writer()
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}

	// sub_topic_id is acceptable as query param (not security-sensitive)
	subTopicId := ctx.Request().Query("sub_topic_id", "global")

	// Sanitize sub_topic_id to prevent key injection in the map
	if len(subTopicId) > 50 {
		subTopicId = "global"
	}

	w := ctx.Response().Writer()
	r := ctx.Request().Origin()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket Upgrade Error:", err)
		return nil
	}

	clientsMu.Lock()
	if clients[userIDStr] == nil {
		clients[userIDStr] = make(map[string]*websocket.Conn)
	}
	clients[userIDStr][subTopicId] = conn
	clientsMu.Unlock()

	log.Printf("WebSocket Client Connected: User=%s, SubTopic=%s\n", userIDStr, subTopicId)

	go func() {
		defer func() {
			clientsMu.Lock()
			if clients[userIDStr] != nil {
				if clients[userIDStr][subTopicId] == conn {
					delete(clients[userIDStr], subTopicId)
					if len(clients[userIDStr]) == 0 {
						delete(clients, userIDStr)
					}
				}
			}
			clientsMu.Unlock()
			conn.Close()
			log.Printf("WebSocket Client Disconnected: User=%s, SubTopic=%s\n", userIDStr, subTopicId)
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()

	return nil
}

// BroadcastMessage sends a message to a specific user and sub-topic.
func BroadcastMessage(userId string, subTopicId string, message []byte) error {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if subTopicId == "" {
		subTopicId = "global"
	}

	if userId != "0" {
		if userClients, ok := clients[userId]; ok {
			if conn, ok := userClients[subTopicId]; ok {
				return conn.WriteMessage(websocket.TextMessage, message)
			}
		}
	}

	for _, userClients := range clients {
		if conn, ok := userClients[subTopicId]; ok {
			_ = conn.WriteMessage(websocket.TextMessage, message)
		}
	}

	return nil
}
