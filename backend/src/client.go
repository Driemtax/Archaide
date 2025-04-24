package main

import (
	"encoding/json"
	"log"
	"time"

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
	hub *Hub
	// Die WebSocket-Verbindung.
	conn *websocket.Conn
	// Gepufferter Kanal für ausgehende Nachrichten.
	send chan []byte
	// Eindeutige ID für den Client
	id string
	// Der aktuelle Score des Spielers
	score int
	// Das vom Spieler ausgewählte Spiel in der aktuellen Runde
	selectedGame string
}

// readPump pumpt Nachrichten von der WebSocket-Verbindung zum Hub.
// Die Anwendung startet readPump in einer eigenen Goroutine für jede Verbindung.
// Sie stellt sicher, dass höchstens eine Leseoperation pro Verbindung läuft.
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
			break // Beendet die Schleife bei Fehlern (z.B. Verbindungsabbruch)
		}

		// Verarbeite die empfangene Nachricht
		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("error unmarshalling message from client %s: %v", c.id, err)
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
		c.hub.incoming <- hubMsg // Sende an den Hub zur Verarbeitung
	}
}

// writePump pumpt Nachrichten vom Hub zur WebSocket-Verbindung.
// Eine Goroutine, die writePump ausführt, wird für jede Verbindung gestartet. Die
// Anwendung stellt sicher, dass höchstens eine Schreiboperation pro Verbindung läuft,
// indem alle Nachrichten über den `send`-Kanal dieses Clients gesendet werden.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		log.Printf("Client %s writePump closed", c.id)
		// Hinweis: Das Unregister sollte idealerweise vom readPump ausgelöst werden,
		// da Lese-Fehler zuerst auftreten. Ein Fehler hier bedeutet meist,
		// dass die Verbindung bereits weg ist.
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Der Hub hat den Kanal geschlossen.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				log.Printf("Client %s send channel closed by hub", c.id)
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("error getting writer for client %s: %v", c.id, err)
				return
			}
			w.Write(message)

			// Füge alle weiteren Nachrichten in der Warteschlange hinzu.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'}) // Trenne Nachrichten mit Newline, falls gewünscht
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				log.Printf("error closing writer for client %s: %v", c.id, err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("error sending ping to client %s: %v", c.id, err)
				return // Bei Ping-Fehler annehmen, dass die Verbindung tot ist
			}
		}
	}
}

// Helper zum Senden einer strukturierten Nachricht an diesen Client
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

	// Sende die Nachricht nicht-blockierend, um Deadlocks zu vermeiden, falls der send-Puffer voll ist
	select {
	case c.send <- messageBytes:
	default:
		log.Printf("Client %s send buffer full. Dropping message.", c.id)
		// Optional: Schließe die Verbindung, wenn der Puffer dauerhaft voll ist
		// close(c.send) // Vorsicht: Dies würde den writePump beenden
	}
	return nil
}
