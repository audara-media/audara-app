package websocket

import (
	"encoding/json"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn       *websocket.Conn
	done       chan struct{}
	send       chan []byte
	webappURL  string
	token      string
	userID     string
	onKeyPress func(string)
	onStatus   func(bool)
	closed     bool
	mu         sync.Mutex
}

type Message struct {
	Type    string          `json:"type"`
	Command string          `json:"command,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type KeyCodeMessage struct {
	Type    string `json:"type"`
	KeyCode string `json:"keyCode"`
	UserID  string `json:"userId"`
}

func NewClient(webappURL, token, userID string) *Client {
	return &Client{
		done:      make(chan struct{}),
		send:      make(chan []byte, 256),
		webappURL: webappURL,
		token:     token,
		userID:    userID,
	}
}

func (c *Client) SetKeyPressHandler(handler func(string)) {
	c.onKeyPress = handler
}

func (c *Client) SetConnectionStatusHandler(handler func(bool)) {
	c.onStatus = handler
}

func (c *Client) Connect() error {
	if c.conn != nil {
		c.Close()
	}

	c.done = make(chan struct{})
	c.send = make(chan []byte, 256)
	c.closed = false

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
		if c.onStatus != nil {
			c.onStatus(false)
		}
		return err
	}

	c.conn = conn
	if c.onStatus != nil {
		c.onStatus(true)
	}

	go c.readPump()
	go c.writePump()

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
		c.mu.Lock()
		defer c.mu.Unlock()

		if !c.closed {
			c.closed = true
			if c.conn != nil {
				c.conn.Close()
			}
			close(c.done)
			if c.onStatus != nil {
				c.onStatus(false)
			}
		}
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		log.Printf("Received WebSocket message: %s", string(message))

		var msg KeyCodeMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error parsing message: %v", err)
			log.Printf("Message: %s", string(message))
			continue
		}

		if msg.Type == "keyCode" {
			if msg.UserID != c.userID {
				log.Printf("Received message from different user ID: %s (expected: %s)", msg.UserID, c.userID)
				continue
			}

			if c.onKeyPress != nil {
				c.onKeyPress(msg.KeyCode)
			}
		}
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		if c.conn != nil {
			c.conn.Close()
		}
		close(c.done)
	}
}
