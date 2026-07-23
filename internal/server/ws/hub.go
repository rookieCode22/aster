// Package ws provides WebSocket streaming for real-time chat.
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

// Message represents a chat message sent via WebSocket.
type Message struct {
	Type    string          `json:"type"`    // "chat", "ping", "pong"
	Payload json.RawMessage `json:"payload"` // JSON-encoded payload
}

// StreamEvent is sent from the server to the client.
type StreamEvent struct {
	Type    string `json:"type"`    // "phase", "thinking", "tool_call", "tool_result", "chunk", "done", "error"
	Payload any    `json:"payload"` // arbitrary JSON-serializable
}

// Client is a single WebSocket connection.
type Client struct {
	hub       *Hub
	conn      wsConn
	send      chan []byte
	sessionID string
	userID    string
}

// wsConn abstracts the WebSocket connection for testability.
type wsConn interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	Close() error
}

// Hub manages all WebSocket connections and message routing.
type Hub struct {
	mu       sync.RWMutex
	clients  map[*Client]bool
	pending  chan func() // serialized operations
	stopped  chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]bool),
		pending: make(chan func(), 256),
		stopped: make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case fn := <-h.pending:
			fn()
		case <-h.stopped:
			return
		}
	}
}

func (h *Hub) Stop() {
	close(h.stopped)
}

func (h *Hub) Register(client *Client) {
	h.pending <- func() {
		h.mu.Lock()
		h.clients[client] = true
		h.mu.Unlock()
		log.Printf("[ws] client connected (session=%s user=%s)", client.sessionID, client.userID)
	}
}

func (h *Hub) Unregister(client *Client) {
	h.pending <- func() {
		h.mu.Lock()
		if _, ok := h.clients[client]; ok {
			delete(h.clients, client)
			close(client.send)
		}
		h.mu.Unlock()
		log.Printf("[ws] client disconnected (session=%s)", client.sessionID)
	}
}

// SendEvent sends a typed event to all clients in a session.
func (h *Hub) SendEvent(sessionID string, event StreamEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.pending <- func() {
		h.mu.RLock()
		defer h.mu.RUnlock()
		for client := range h.clients {
			if client.sessionID == sessionID {
				select {
				case client.send <- data:
				default:
					// client too slow, skip
				}
			}
		}
	}
}

// HandleUpgrade is the HTTP handler for WebSocket upgrade requests.
func (h *Hub) HandleUpgrade(w http.ResponseWriter, r *http.Request, sessionID, userID string) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := &Client{
		hub:       h,
		conn:      &gorillaConn{conn},
		send:      make(chan []byte, 64),
		sessionID: sessionID,
		userID:    userID,
	}

	h.Register(client)

	go client.writePump()
	go client.readPump()

	return nil
}
