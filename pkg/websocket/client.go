package websocket

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn      *websocket.Conn
	done      chan struct{}
	send      chan []byte
	webappURL string
	token     string
}

type Message struct {
	Type    string          `json:"type"`
	Command string          `json:"command,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func NewClient(webappURL, token string) *Client {
	return &Client{
		done:      make(chan struct{}),
		send:      make(chan []byte, 256),
		webappURL: webappURL,
		token:     token,
	}
}

func (c *Client) Connect() error {
	u, err := url.Parse(c.webappURL)
	if err != nil {
		return err
	}
	u.Scheme = "ws"
	u.Path = "/_ws/"

	header := make(map[string][]string)
	header["Authorization"] = []string{"Bearer " + c.token}

	dialer := websocket.Dialer{
		HandshakeTimeout: 45 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), header)
	if err != nil {
		return err
	}

	c.conn = conn

	// Start goroutines for reading and writing
	go c.readPump()
	go c.writePump()

	// Send initial ping message
	pingMsg := Message{
		Type:    "command",
		Command: "ping",
	}

	msgBytes, err := json.Marshal(pingMsg)
	if err != nil {
		return err
	}

	c.send <- msgBytes

	return nil
}

func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
		close(c.done)
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// Log the received message
		log.Printf("Received WebSocket message: %s", string(message))
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message := <-c.send:
			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *Client) Close() {
	close(c.done)
	c.conn.Close()
}
