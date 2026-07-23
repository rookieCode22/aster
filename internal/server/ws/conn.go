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

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		// Route incoming chat messages to the handler
		c.hub.pending <- func() {
			// Handled by the session's chat loop
		}
	}
}
