package tmpchat

import (
	_ "embed"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

//go:embed tmpchat.html
var chatPage []byte

// Message type constants
const (
	MsgTypeOnline  = "online"
	MsgTypeOffline = "offline"
	MsgTypeChat    = "chat"
	MsgTypeSystem  = "system"
)

// Message structure
type Message struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	Username  string `json:"username,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
	OnlineNum int    `json:"online_num"`
}

// Client represents a connected user
type Client struct {
	server    *ChatServer
	conn      *websocket.Conn
	send      chan Message
	username  string
	closeLock sync.Once
}

// ChatServer maintains the set of active clients
// and broadcasts messages to all clients
type ChatServer struct {
	logger *zap.Logger

	mu      sync.RWMutex
	clients map[*Client]bool

	online    chan *Client
	offline   chan *Client
	broadcast chan Message
	closeCh   chan struct{}
}

// NewChatServer creates a new server
func NewChatServer() *ChatServer {
	return &ChatServer{
		broadcast: make(chan Message, 100),
		online:    make(chan *Client),
		offline:   make(chan *Client),
		clients:   make(map[*Client]bool),
	}
}

// Run starts the ChatServer
func (s *ChatServer) Run() {
	for {
		select {
		case <-s.closeCh:
			return
		case client := <-s.online:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()

			// Broadcast user joined message
			s.broadcast <- Message{
				Type:      MsgTypeOnline,
				Username:  client.username,
				Timestamp: time.Now().Unix(),
				OnlineNum: s.getOnlineCount(),
			}

		case client := <-s.offline:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)

				// Broadcast user left message
				s.broadcast <- Message{
					Type:      MsgTypeOffline,
					Username:  client.username,
					Timestamp: time.Now().Unix(),
					OnlineNum: len(s.clients),
				}
			}
			s.mu.Unlock()

		case message := <-s.broadcast:
			s.broadcastMessage(message)
		}
	}
}

func (s *ChatServer) broadcastMessage(message Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for client := range s.clients {
		select {
		case client.send <- message:
		default:
			client.closeOnce()
		}
	}
}

// Close shutdown the ChatServer
func (s *ChatServer) Close() {
	s.logger.Info("ChatServer is shutting down")
	closeMsg := Message{
		Type:      MsgTypeSystem,
		Content:   "Server is shutting down",
		Timestamp: time.Now().Unix(),
		OnlineNum: 0,
	}
	s.broadcast <- closeMsg
	for {
		select {
		case message := <-s.broadcast:
			s.broadcastMessage(message)
		default:
			close(s.closeCh)
			return
		}
	}
}

// getOnlineCount returns the number of connected clients
func (s *ChatServer) getOnlineCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// serveWebsocket handles websocket requests from clients
func (s *ChatServer) serveWebsocket(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("u")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		server:   s,
		conn:     conn,
		send:     make(chan Message, 256),
		username: username,
	}

	// Register the client
	select {
	case s.online <- client:
	case <-s.closeCh:
		return
	}

	// Start goroutines for reading and writing messages
	go client.writePump()
	go client.readPump()
}

func (c *Client) closeOnce() {
	c.closeLock.Do(func() {
		c.conn.Close()
		select {
		case c.server.offline <- c:
		case <-c.server.closeCh:
			return
		}
	})
}

// readPump pumps messages from the websocket connection to the server
func (c *Client) readPump() {
	defer c.closeOnce()

	c.conn.SetReadLimit(512 * 1024) // 512KB
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
			}
			break
		}

		// Handle different message types
		switch msg.Type {
		case MsgTypeChat:
			// Broadcast chat message to all clients
			msg.Username = c.username
			msg.Timestamp = time.Now().Unix()
			c.server.broadcast <- msg
		}
	}
}

// writePump pumps messages from the server to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second) // Ping interval
	defer c.closeOnce()
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if !ok {
				// The server closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send the message
			err := c.conn.WriteJSON(message)
			if err != nil {
				log.Printf("Error writing message: %v", err)
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
