package websocket

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Client represents a websocket connection bound to a user.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID int
}

// Hub manages active clients and broadcasts.
type Hub struct {
	register   chan *Client
	unregister chan *Client
	// Map of userID to set of clients
	clientsByUser map[int]map[*Client]bool
}

// NewHub creates and starts a new Hub loop.
func NewHub() *Hub {
	h := &Hub{
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		clientsByUser: make(map[int]map[*Client]bool),
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			set, ok := h.clientsByUser[c.userID]
			if !ok {
				set = make(map[*Client]bool)
				h.clientsByUser[c.userID] = set
			}
			set[c] = true
		case c := <-h.unregister:
			if set, ok := h.clientsByUser[c.userID]; ok {
				if _, exists := set[c]; exists {
					delete(set, c)
					close(c.send)
					if len(set) == 0 {
						delete(h.clientsByUser, c.userID)
					}
				}
			}
		}
	}
}

// NotifyUser sends a payload to all connected clients of a given user.
func (h *Hub) NotifyUser(userID int, payload []byte) {
	if h == nil {
		return
	}
	if set, ok := h.clientsByUser[userID]; ok {
		for c := range set {
			select {
			case c.send <- payload:
			default:
				// Backpressure: drop and disconnect slow clients
				close(c.send)
				delete(set, c)
			}
		}
		if len(set) == 0 {
			delete(h.clientsByUser, userID)
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// ServeWS upgrades HTTP connection to WebSocket and registers the client.
// JWT is not parsed here to avoid duplication; caller must authenticate and set userId in context.
func ServeWS(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt("userId")
		if userID == 0 {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("websocket upgrade failed", "err", err)
			return
		}
		client := &Client{hub: h, conn: conn, send: make(chan []byte, 256), userID: userID}
		h.register <- client

		// Reader goroutine
		go func() {
			defer func() {
				h.unregister <- client
				_ = conn.Close()
			}()
			conn.SetReadLimit(1024)
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			conn.SetPongHandler(func(string) error {
				return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			})
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
			}
		}()

		// Writer loop (same goroutine)
		for msg := range client.send {
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				break
			}
		}
	}
}

// Debug helper to send a ping message to user via HTTP
func (h *Hub) DebugSend(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Query("userId"))
	msg := c.Query("msg")
	h.NotifyUser(uid, []byte(msg))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
