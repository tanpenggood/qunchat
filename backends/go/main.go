package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ReplyTo struct {
	Name     string `json:"name"`
	Text     string `json:"text"`
	SenderId string `json:"senderId"`
}

type Message struct {
	Cmd      string   `json:"cmd,omitempty"`
	Name     string   `json:"name"`
	Text     string   `json:"text,omitempty"`
	Time     int64    `json:"time,omitempty"`
	SenderId string   `json:"senderId,omitempty"`
	Online   int      `json:"online,omitempty"`
	ReplyTo  *ReplyTo `json:"replyTo,omitempty"`
}

type userSession struct {
	name  string
	count int
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	name     string
	senderId string
}

type Hub struct {
	clients      map[*Client]bool
	userSessions map[string]*userSession
	broadcast    chan Message
	register     chan *Client
	unregister   chan *Client
	mu           sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:      make(map[*Client]bool),
		userSessions: make(map[string]*userSession),
		broadcast:    make(chan Message, 256),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket connected")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				sid := client.senderId
				if sid != "" {
					if s, ok := h.userSessions[sid]; ok {
						s.count--
						if s.count <= 0 {
							delete(h.userSessions, sid)
							h.mu.Unlock()
							h.broadcastSystem(client.name + " 离开了群聊")
							log.Printf("User left: %s (online: %d)", client.name, len(h.userSessions))
						} else {
							h.mu.Unlock()
							log.Printf("Tab closed: %s (still %d tabs open)", client.name, s.count)
						}
					} else {
						h.mu.Unlock()
					}
				} else {
					h.mu.Unlock()
				}
			} else {
				h.mu.Unlock()
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			message.Online = len(h.userSessions)
			data, _ := json.Marshal(message)
			for client := range h.clients {
				select {
				case client.send <- data:
				default:
					go func(c *Client) { h.unregister <- c }(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) GetOnlineUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	names := make([]string, 0, len(h.userSessions))
	for _, s := range h.userSessions {
		names = append(names, s.name)
	}
	return names
}

func (h *Hub) broadcastSystem(text string) {
	h.broadcast <- Message{Name: "系统", Text: text, Time: time.Now().UnixMilli()}
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(8192)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if c.name == "" {
			if msg.Name == "" || msg.SenderId == "" {
				continue
			}
			c.name = msg.Name
			c.senderId = msg.SenderId
			h := c.hub
			h.mu.Lock()
			if s, ok := h.userSessions[c.senderId]; ok {
				s.count++
				online := len(h.userSessions)
				h.mu.Unlock()
				welcome, _ := json.Marshal(Message{Name: "系统", Text: "已连接", Online: online})
				c.send <- welcome
			} else {
				h.userSessions[c.senderId] = &userSession{name: c.name, count: 1}
				h.mu.Unlock()
				h.broadcastSystem(c.name + " 加入了群聊")
			}
			continue
		}

		if msg.Cmd == "rename" && msg.Name != "" {
			oldName := c.name
			c.name = msg.Name
			h := c.hub
			h.mu.Lock()
			if s, ok := h.userSessions[c.senderId]; ok {
				s.name = msg.Name
			}
			for client := range h.clients {
				if client.senderId == c.senderId {
					client.name = msg.Name
				}
			}
			h.mu.Unlock()
			h.broadcastSystem(oldName + " 改名为 " + c.name)
			continue
		}

		if msg.Cmd == "whoisonline" {
			users := c.hub.GetOnlineUsers()
			resp, _ := json.Marshal(map[string]any{
				"type":  "online_list",
				"users": users,
			})
			c.send <- resp
			continue
		}

		if msg.Text == "" {
			continue
		}
		msg.Name = c.name
		msg.Time = time.Now().UnixMilli()
		c.hub.broadcast <- msg
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return port
}

func main() {
	hub := NewHub()
	go hub.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	http.Handle("/", http.FileServer(http.Dir("./frontend")))

	port := getPort()
	log.Printf("Chat server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
