package coms

import (
	"encoding/json"
	"log"
	"time"

	"github.com/Driemtax/Archaide/internal/message"
	"github.com/gorilla/websocket"
)

const (
	// Zeit, die für das Schreiben einer Nachricht an den Peer erlaubt ist.
	writeWait = 10 * time.Second
	// Zeit, die für das Lesen der nächsten Pong-Nachricht vom Peer erlaubt ist.
	pongWait = 60 * time.Second
	// Sende Pings an den Peer mit diesem Intervall. Muss kleiner als pongWait sein.
	pingPeriod = (pongWait * 9) / 10
	// Maximale Nachrichtengröße, die vom Peer erlaubt ist.
	maxMessageSize = 512
)

// Client ist eine Zwischeninstanz zwischen der WebSocket-Verbindung und dem Hub.
type Client struct {
	Hub *Hub
	// Die WebSocket-Verbindung.
	Conn *websocket.Conn
	// Gepufferter Kanal für ausgehende Nachrichten.
	Send chan []byte
	// Eindeutige ID für den Client
	Id string
	// Der aktuelle Score des Spielers
	Score int
	// Das vom Spieler ausgewählte Spiel in der aktuellen Runde
	SelectedGame string
}

// readPump pumpt Nachrichten von der WebSocket-Verbindung zum Hub.
// Die Anwendung startet readPump in einer eigenen Goroutine für jede Verbindung.
// Sie stellt sicher, dass höchstens eine Leseoperation pro Verbindung läuft.
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
			break // Beendet die Schleife bei Fehlern (z.B. Verbindungsabbruch)
		}

		// Verarbeite die empfangene Nachricht
		var msg message.Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("error unmarshalling message from client %s: %v", c.Id, err)
			// Sende ggf. eine Fehlermeldung zurück an den Client
			continue
		}

		// Leite die Nachricht zur Verarbeitung an den Hub weiter
		// Der Hub kann dann basierend auf msg.Type entscheiden, was zu tun ist.
		// Wir fügen die Client-ID hinzu, damit der Hub weiß, von wem die Nachricht kam.
		hubMsg := HubMessage{
			client:  c,
			message: msg,
		}
		c.Hub.Incoming <- hubMsg // Sende an den Hub zur Verarbeitung
	}
}

// writePump pumpt Nachrichten vom Hub zur WebSocket-Verbindung.
// Eine Goroutine, die writePump ausführt, wird für jede Verbindung gestartet. Die
// Anwendung stellt sicher, dass höchstens eine Schreiboperation pro Verbindung läuft,
// indem alle Nachrichten über den `send`-Kanal dieses Clients gesendet werden.
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
				// Der Hub hat den Kanal geschlossen.
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

// Helper zum Senden einer strukturierten Nachricht an diesen Client
func (c *Client) sendMessage(msgType string, payload interface{}) error {
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

	// Sende die Nachricht nicht-blockierend, um Deadlocks zu vermeiden, falls der send-Puffer voll ist
	select {
	case c.Send <- messageBytes:
	default:
		log.Printf("Client %s send buffer full. Dropping message.", c.Id)
		// Optional: Schließe die Verbindung, wenn der Puffer dauerhaft voll ist
		// close(c.send) // Vorsicht: Dies würde den writePump beenden
	}
	return nil
}
