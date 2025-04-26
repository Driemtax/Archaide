package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Client is an intermediate instance between the WebSocket connection and the Hub.
type Client struct {
	hub *Hub
	// The WebSocket connection.
	conn *websocket.Conn
	// Buffered channel of outbound messages.
	send chan []byte
	// Unique ID for the client.
	id string
	// The player's current score.
	score int
	// The game selected by the player in the current round.
	selectedGame string
}

// readPump pumps messages from the WebSocket connection to the Hub.
// The application runs readPump in its own goroutine for each connection.
// It ensures that at most one read operation is performed per connection.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		log.Printf("Client %s disconnected (readPump closed)", c.id)
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message for client %s: %v", c.id, err)
			}
			break // Exits the loop on errors (e.g., connection closed)
		}

		// Process the received message
		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("error unmarshalling message from client %s: %v", c.id, err)
			// Optionally send an error message back to the client
			continue
		}

		// Forward the message to the Hub for processing
		// The Hub can then decide what to do based on msg.Type.
		// We add the client ID so the Hub knows who sent the message.
		hubMsg := HubMessage{
			client:  c,
			message: msg,
		}
		c.hub.incoming <- hubMsg // Send to the Hub for processing
	}
}

// writePump pumps messages from the Hub to the WebSocket connection.
// A goroutine running writePump is started for each connection. The
// application ensures that at most one write operation is performed per connection
// by sending all messages through the client's `send` channel.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		log.Printf("Client %s writePump closed", c.id)
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The Hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				log.Printf("Client %s send channel closed by hub", c.id)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("error writing message to client %s: %v", c.id, err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("error sending ping to client %s: %v", c.id, err)
				return
			}
		}
	}
}

// Helper for sending a structured message to this client
func (c *Client) sendMessage(msgType string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling payload for client %s: %v", c.id, err)
		return err
	}
	message := Message{
		Type:    msgType,
		Payload: json.RawMessage(payloadBytes),
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshalling message for client %s: %v", c.id, err)
		return err
	}

	// Send the message non-blocking to avoid deadlocks if the send buffer is full
	select {
	case c.send <- messageBytes:
	default:
		log.Printf("Client %s send buffer full. Dropping message.", c.id)
		// Optionally: Close the connection if the buffer is permanently full
		// close(c.send) // Caution: This would terminate writePump
	}
	return nil
}
