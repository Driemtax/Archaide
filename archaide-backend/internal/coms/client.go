package coms

import (
	"encoding/json"
	"log"
	"time"

	"github.com/Driemtax/Archaide/internal/message"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Client struct {
	Hub          *Hub
	Conn         *websocket.Conn
	Send         chan []byte
	Id           string
	Score        int
	SelectedGame string
}

// ReadPump transfers messages from the WebSocket to the Hub.
// Runs in a separate goroutine for each connection, ensuring only one
// read operation occurs per connection at a time.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
		log.Printf("Client %s disconnected (readPump closed)", c.Id)
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message for client %s: %v", c.Id, err)
			}
			break
		}

		var msg message.Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("error unmarshalling message from client %s: %v", c.Id, err)
			continue
		}

		hubMsg := HubMessage{
			client:  c,
			message: msg,
		}
		c.Hub.Incoming <- hubMsg
	}
}

// WritePump transfers messages from the Hub to the WebSocket connection.
// Ensures that there is at most one writer to a connection by
// multiplexing all messages through the client's Send channel.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
		log.Printf("Client %s writePump closed", c.Id)
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				log.Printf("Client %s send channel closed by hub", c.Id)
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("error writing message to client %s: %v", c.Id, err)
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("error sending ping to client %s: %v", c.Id, err)
				return
			}
		}
	}
}

// sendMessage formats and sends a structured message to the client
// Uses non-blocking send to prevent deadlocks if buffer is full
func (c *Client) sendMessage(msgType string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling payload for client %s: %v", c.Id, err)
		return err
	}
	message := message.Message{
		Type:    msgType,
		Payload: json.RawMessage(payloadBytes),
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshalling message for client %s: %v", c.Id, err)
		return err
	}

	select {
	case c.Send <- messageBytes:
	default:
		log.Printf("Client %s send buffer full. Dropping message.", c.Id)
	}
	return nil
}
