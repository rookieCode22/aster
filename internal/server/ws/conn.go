package ws

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type gorillaConn struct {
	*websocket.Conn
}

func (c *gorillaConn) ReadMessage() (int, []byte, error) {
	return c.Conn.ReadMessage()
}

func (c *gorillaConn) WriteMessage(t int, data []byte) error {
	return c.Conn.WriteMessage(t, data)
}

func (c *gorillaConn) Close() error {
	return c.Conn.Close()
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for data := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[ws] write error: %v", err)
			return
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[ws] read error: %v", err)
			}
			return
		}

		// The frontend sends {type, content}. Parse leniently: prefer a top-level
		// "content" field; fall back to a JSON payload with a content field.
		var raw struct {
			Type    string          `json:"type"`
			Content string          `json:"content"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}

		if raw.Type == "ping" {
			continue
		}

		content := raw.Content
		if content == "" && len(raw.Payload) > 0 {
			var p struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal(raw.Payload, &p); err == nil {
				content = p.Content
			}
		}
		if content == "" {
			continue
		}

		// Route incoming chat message to the agent engine. dispatchChat runs the
		// handler in its own goroutine so this read loop stays responsive.
		c.hub.dispatchChat(c.sessionID, c.userID, content)
	}
}
