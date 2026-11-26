package server

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 45 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // LAN multiplayer across local IPs
	},
}

// Client wraps a single WebSocket connection.
type Client struct {
	conn      *websocket.Conn
	sendCh    chan ServerMsg
	room      *Room
	closeOnce sync.Once
}

// Send queues a message to be written to the client's WebSocket.
func (c *Client) Send(msg ServerMsg) {
	select {
	case c.sendCh <- msg:
	default:
		// Drop if client buffer is full
	}
}

// Close closes the underlying WebSocket connection.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.sendCh)
		_ = c.conn.Close()
	})
}

// ServeWS upgrades an HTTP request to a WebSocket and attaches it to the Room.
func ServeWS(room *Room, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}
	client := &Client{
		conn:   conn,
		sendCh: make(chan ServerMsg, 32),
		room:   room,
	}
	room.RegisterClient(client)
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.room.HandleDisconnect(c)
		c.Close()
	}()

	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg ClientMsg
		if err := c.conn.ReadJSON(&msg); err != nil {
			break
		}
		c.room.HandleMessage(c, msg)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.sendCh:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
