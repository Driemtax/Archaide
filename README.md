# Archaide
Archaide is a multiplayer arcade plattform to battle your friends

**Ziele:**

1.  Ein Go-Server, der WebSocket-Verbindungen akzeptiert.
2.  Eine zentrale "Lobby" (oft als "Hub" bezeichnet), die alle verbundenen Spieler verwaltet.
3.  Jeder Spieler (`Client`) hat eine eindeutige ID und einen Score.
4.  Spieler können eine Nachricht senden, um ein Spiel auszuwählen.
5.  Wenn alle Spieler gewählt haben, wählt der Server zufällig ein Spiel aus.
6.  Die Struktur sollte erweiterbar sein, um später tatsächliche Spiellogik und Score-Updates hinzuzufügen.

**Technologien:**

*   Go (Programmiersprache)
*   `net/http` (Go Standard Library für den HTTP-Server)
*   `github.com/gorilla/websocket` (Populäre WebSocket-Bibliothek für Go)

**Struktur (Dateien):**

*   `main.go`: Der Einstiegspunkt. Startet den HTTP-Server und den Hub.
*   `hub.go`: Definiert die zentrale Logik der Lobby (Hub), die Clients verwaltet und Nachrichten broadcastet.
*   `client.go`: Definiert die Struktur und Logik für einen einzelnen verbundenen Spieler (Client).
*   `message.go` (Optional, aber empfohlen): Definiert die Strukturen für Nachrichten, die über WebSockets gesendet werden (z.B. im JSON-Format).

---

**1. `message.go` (Nachrichtenstrukturen)**

Wir definieren, wie unsere Nachrichten aussehen. JSON ist hier ein gängiges Format.

```go
package main

import "encoding/json"

// Message repräsentiert eine generische Nachricht, die über WebSocket gesendet wird.
// Der 'Type' hilft dem Server oder Client zu verstehen, wie der 'Payload' zu interpretieren ist.
type Message struct {
	Type    string          `json:"type"`    // z.B. "update_lobby", "select_game", "error", "welcome"
	Payload json.RawMessage `json:"payload"` // Die eigentlichen Daten, abhängig vom Typ
}

// Payload-Strukturen (Beispiele)

// WelcomeMessage enthält die ID des neuen Clients und die Liste der verfügbaren Spiele
type WelcomeMessage struct {
	ClientID     string   `json:"clientId"`
	CurrentGames []string `json:"currentGames"`
}

// LobbyUpdateMessage enthält den aktuellen Zustand der Lobby (Spieler und ihre Scores)
type LobbyUpdateMessage struct {
	Players map[string]int `json:"players"` // Map von ClientID zu Score
}

// SelectGamePayload wird vom Client gesendet, wenn er ein Spiel auswählt
type SelectGamePayload struct {
	Game string `json:"game"`
}

// GameSelectedMessage wird an alle gesendet, wenn ein Spiel ausgewählt wurde
type GameSelectedMessage struct {
    SelectedGame string `json:"selectedGame"`
}

// ErrorMessage wird bei Fehlern gesendet
type ErrorMessage struct {
	Message string `json:"message"`
}
```

---

**2. `client.go` (Logik für einen Spieler)**

Jede WebSocket-Verbindung wird durch eine `Client`-Instanz repräsentiert.

```go
package main

import (
	"log"
	"time"
    "encoding/json"

	"github.com/gorilla/websocket"
	"github.com/google/uuid" // Für eindeutige IDs
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
            client: c,
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
        Type: msgType,
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
```

---

**3. `hub.go` (Die zentrale Lobby)**

Der Hub verwaltet alle Clients und die Spiellogik (wie die Auswahl).

```go
package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"
)

// HubMessage ist eine Wrapper-Struktur, um Nachrichten zusammen mit dem sendenden Client an den Hub zu übergeben.
type HubMessage struct {
    client *Client
    message Message
}

// Hub verwaltet den Satz aktiver Clients und broadcastet Nachrichten an sie.
type Hub struct {
	// Registrierte Clients. Die Keys sind die Client-Pointer, der Wert ist immer true.
	// Oder: map[string]*Client für einfacheren Zugriff per ID
	clients map[*Client]bool

	// Eingehende Nachrichten von den Clients.
    incoming chan HubMessage

	// Registrierungsanfragen von Clients.
	register chan *Client

	// Deregistrierungsanfragen von Clients.
	unregister chan *Client

	// Liste der verfügbaren Spiele
	availableGames []string

    // Spielauswahlen der aktuellen Runde (Client -> Spielname)
    currentGameSelections map[*Client]string
}

func newHub() *Hub {
	return &Hub{
		incoming:   make(chan HubMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
        availableGames: []string{"Asteroids", "Pong", "Space Invaders"}, // Beispielspiele
        currentGameSelections: make(map[*Client]string),
	}
}

func (h *Hub) run() {
	log.Println("Hub is running...")
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("Client %s registered. Total clients: %d", client.id, len(h.clients))

            // Sende eine Willkommensnachricht an den neuen Client
            welcomePayload := WelcomeMessage{
                ClientID:     client.id,
                CurrentGames: h.availableGames,
            }
            client.sendMessage("welcome", welcomePayload)

			// Sende den aktuellen Lobby-Status an alle Clients
			h.broadcastLobbyUpdate()

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
                delete(h.currentGameSelections, client) // Entferne auch die Auswahl
				close(client.send) // Schließe den Send-Kanal des Clients
				log.Printf("Client %s unregistered. Total clients: %d", client.id, len(h.clients))
				// Sende den aktualisierten Lobby-Status an die verbleibenden Clients
				h.broadcastLobbyUpdate()
                // Überprüfe nach dem Verlassen, ob nun alle gewählt haben (falls jemand geht, während abgestimmt wird)
                h.checkAllPlayersSelectedGame()
			}

		case hubMsg := <-h.incoming:
            // Verarbeite die Nachricht vom Client
            h.handleIncomingMessage(hubMsg.client, hubMsg.message)
		}
	}
}

// Verarbeitet eingehende Nachrichten von einem Client
func (h *Hub) handleIncomingMessage(client *Client, msg Message) {
    log.Printf("Received message type '%s' from client %s", msg.Type, client.id)
    switch msg.Type {
    case "select_game":
        var payload SelectGamePayload
        if err := json.Unmarshal(msg.Payload, &payload); err != nil {
            log.Printf("Error unmarshalling select_game payload from %s: %v", client.id, err)
            client.sendMessage("error", ErrorMessage{Message: "Invalid select_game payload"})
            return
        }

        // Validieren, ob das Spiel gültig ist
        isValidGame := false
        for _, game := range h.availableGames {
            if game == payload.Game {
                isValidGame = true
                break
            }
        }
        if !isValidGame {
             log.Printf("Client %s selected invalid game: %s", client.id, payload.Game)
             client.sendMessage("error", ErrorMessage{Message: "Invalid game selected"})
             return
        }

        // Speichere die Auswahl des Spielers
        h.currentGameSelections[client] = payload.Game
        client.selectedGame = payload.Game // Auch im Client speichern
        log.Printf("Client %s selected game: %s", client.id, payload.Game)

        // Sende ggf. eine Bestätigung oder Update an Clients (optional)
        // client.sendMessage("game_selection_received", ...)

        // Überprüfe, ob alle Spieler gewählt haben
        h.checkAllPlayersSelectedGame()

    // Hier könnten weitere Nachrichten-Typen behandelt werden (z.B. Chat)
    default:
        log.Printf("Received unhandled message type '%s' from client %s", msg.Type, client.id)
    }
}

// Überprüft, ob alle verbundenen Spieler ein Spiel für die aktuelle Runde ausgewählt haben
func (h *Hub) checkAllPlayersSelectedGame() {
    if len(h.clients) == 0 {
        return // Niemand da, nichts zu tun
    }

    allSelected := true
    for client := range h.clients {
        if _, ok := h.currentGameSelections[client]; !ok {
            allSelected = false
            break
        }
    }

    if allSelected {
        log.Printf("All %d players have selected a game. Determining winner...", len(h.clients))
        h.selectAndAnnounceGame()
        // Setze die Auswahlen für die nächste Runde zurück
        h.currentGameSelections = make(map[*Client]string)
        for client := range h.clients {
            client.selectedGame = "" // Auch im Client zurücksetzen
        }
    } else {
        log.Printf("%d out of %d players have selected a game.", len(h.currentGameSelections), len(h.clients))
        // Optional: Sende ein Update, wer noch nicht gewählt hat
    }
}

// Wählt zufällig ein Spiel basierend auf den Auswahlen aus und kündigt es an
func (h *Hub) selectAndAnnounceGame() {
    if len(h.currentGameSelections) == 0 {
        log.Println("No selections made, cannot select a game.")
        return
    }

    // Einfache zufällige Auswahl aus den gewählten Spielen
    // TODO: Implementiere die gewichtete Auswahl basierend auf der Häufigkeit der Auswahl
    selections := []string{}
    for _, gameName := range h.currentGameSelections {
        selections = append(selections, gameName)
    }

    rand.Seed(time.Now().UnixNano()) // Seed für Zufallszahlengenerator
    randomIndex := rand.Intn(len(selections))
    selectedGame := selections[randomIndex]

    log.Printf("Randomly selected game: %s", selectedGame)

    // Sende das Ergebnis an alle Clients
    announcementPayload := GameSelectedMessage{SelectedGame: selectedGame}
    h.broadcastMessage("game_selected", announcementPayload)

    // --- Hier würde die Logik zum Starten des Spiels beginnen ---
    // Zum Beispiel: Sende eine "start_game" Nachricht mit Details
    // h.startGame(selectedGame)
}


// broadcastLobbyUpdate sendet den aktuellen Spielerstatus (ID und Score) an alle Clients.
func (h *Hub) broadcastLobbyUpdate() {
	playerScores := make(map[string]int)
	for client := range h.clients {
		playerScores[client.id] = client.score
	}
	payload := LobbyUpdateMessage{Players: playerScores}

    h.broadcastMessage("update_lobby", payload)
}

// broadcastMessage sendet eine Nachricht an alle verbundenen Clients.
func (h *Hub) broadcastMessage(msgType string, payload interface{}) {
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        log.Printf("Error marshalling payload for broadcast: %v", err)
        return
    }
    message := Message{
        Type: msgType,
        Payload: json.RawMessage(payloadBytes),
    }
    messageBytes, err := json.Marshal(message)
     if err != nil {
        log.Printf("Error marshalling message for broadcast: %v", err)
        return
    }

	log.Printf("Broadcasting message type '%s' to %d clients", msgType, len(h.clients))
	for client := range h.clients {
		select {
		case client.send <- messageBytes:
		default:
			// Wenn der Send-Kanal blockiert oder geschlossen ist,
            // gehen wir davon aus, dass der Client langsam oder getrennt ist.
            // Wir entfernen ihn hier nicht direkt, das sollte der readPump/writePump tun.
			log.Printf("Could not send broadcast to client %s (send buffer full or closed)", client.id)
            // Optional: Man könnte den Client hier aggressiver entfernen:
            // close(client.send)
			// delete(h.clients, client)
		}
	}
}


// --- Platzhalter für Spiellogik ---

// Diese Funktion würde aufgerufen, nachdem ein Spiel ausgewählt wurde.
// Sie könnte z.B. den Zustand ändern oder spezifische Nachrichten senden.
// func (h *Hub) startGame(gameName string) {
//     log.Printf("Starting game: %s", gameName)
//     // Sende "start_game" Nachricht an alle Clients
//     // Ändere ggf. den Hub- oder Client-Status
// }

// Diese Funktion würde (z.B. durch eine Nachricht vom Spielmodul)
// aufgerufen, um Scores zu aktualisieren.
func (h *Hub) updateScores(scores map[string]int) { // map[ClientID]scoreDelta
    log.Println("Updating scores...")
    for clientID, delta := range scores {
        // Finde den Client anhand der ID (besser, wenn h.clients eine map[string]*Client wäre)
        var targetClient *Client = nil
        for c := range h.clients {
            if c.id == clientID {
                targetClient = c
                break
            }
        }

        if targetClient != nil {
            targetClient.score += delta
             log.Printf("Score updated for %s: new score %d", targetClient.id, targetClient.score)
        } else {
            log.Printf("Could not find client %s to update score", clientID)
        }
    }
    // Sende die neuen Scores an alle
    h.broadcastLobbyUpdate()
}
```

---

**4. `main.go` (Der Server)**

Startet alles und definiert den WebSocket-Endpunkt.

```go
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", ":8080", "http service address")

// upgrader konvertiert HTTP-Verbindungen zu WebSockets.
// CheckOrigin wird hier unsicher konfiguriert, um alle Ursprünge zu erlauben (für Entwicklung).
// In Produktion solltest du dies einschränken!
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// serveWs behandelt WebSocket-Anfragen vom Peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	log.Println("Client connected from:", conn.RemoteAddr())

	// Erstelle einen neuen Client für diese Verbindung
	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256), // Puffergröße für ausgehende Nachrichten
		id:   uuid.New().String(),    // Eindeutige ID generieren
		score: 0,                     // Start-Score
        selectedGame: "",             // Noch kein Spiel ausgewählt
	}
	client.hub.register <- client // Registriere den Client beim Hub

	// Starte die Pump-Goroutinen für diesen Client
    // Diese Goroutinen laufen, bis die Verbindung geschlossen wird oder ein Fehler auftritt.
	go client.writePump()
	go client.readPump()

    // readPump kümmert sich darum, den Client beim Hub abzumelden, wenn die Verbindung endet.
}

func main() {
	flag.Parse()
	hub := newHub() // Erstelle den zentralen Hub
	go hub.run()    // Starte den Hub in einer eigenen Goroutine

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r) // Behandle WebSocket-Verbindungen
	})

    // Optional: Füge einen einfachen HTTP-Handler hinzu, um eine Test-HTML-Seite auszuliefern
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" {
            http.NotFound(w, r)
            return
        }
        http.ServeFile(w, r, "lobby.html") // Annahme: Es gibt eine lobby.html im selben Verzeichnis
    })


	log.Printf("Server starting on %s", *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
```

---

**5. `lobby.html` (Einfaches Frontend zum Testen)**

Eine sehr einfache HTML-Seite mit JavaScript, um sich mit dem Server zu verbinden und Nachrichten zu senden/empfangen.

```html
<!DOCTYPE html>
<html>
<head>
    <title>Go WebSocket Lobby</title>
    <style>
        body { font-family: sans-serif; }
        #log { height: 300px; width: 500px; border: 1px solid #ccc; overflow-y: scroll; margin-bottom: 10px; padding: 5px; }
        .player { margin-bottom: 5px; }
        .player strong { display: inline-block; min-width: 150px;}
        button { margin-right: 5px;}
    </style>
</head>
<body>
    <h1>WebSocket Lobby</h1>
    <div id="log"></div>
    <div id="status">Connecting...</div>
    <div id="players"><h2>Players</h2></div>
    <div id="games"><h2>Select a Game</h2></div>
    <div id="selected-game"></div>

    <script>
        const logDiv = document.getElementById('log');
        const statusDiv = document.getElementById('status');
        const playersDiv = document.getElementById('players');
        const gamesDiv = document.getElementById('games');
        const selectedGameDiv = document.getElementById('selected-game');
        let ws;
        let myClientId = '';
        let availableGames = [];

        function logMessage(message) {
            const p = document.createElement('p');
            p.textContent = message;
            logDiv.appendChild(p);
            logDiv.scrollTop = logDiv.scrollHeight; // Auto-scroll
        }

        function connect() {
            const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const host = window.location.host; // Nimmt Host und Port von der aktuellen Seite
            ws = new WebSocket(`${proto}//${host}/ws`);

            ws.onopen = () => {
                statusDiv.textContent = 'Connected';
                logMessage('WebSocket connection opened.');
            };

            ws.onmessage = (event) => {
                logMessage(`<- Received: ${event.data}`);
                try {
                    const message = JSON.parse(event.data);
                    handleMessage(message);
                } catch (e) {
                    logMessage(`Error parsing message: ${e}`);
                }
            };

            ws.onclose = () => {
                statusDiv.textContent = 'Disconnected. Attempting to reconnect...';
                logMessage('WebSocket connection closed. Reconnecting in 3 seconds...');
                ws = null;
                setTimeout(connect, 3000); // Versuche erneut zu verbinden
            };

            ws.onerror = (error) => {
                statusDiv.textContent = 'Connection Error';
                logMessage(`WebSocket error: ${error.message || 'Unknown error'}`);
                // onclose wird normalerweise auch ausgelöst
            };
        }

        function handleMessage(message) {
            switch (message.type) {
                case 'welcome':
                    myClientId = message.payload.clientId;
                    availableGames = message.payload.currentGames || [];
                    statusDiv.textContent = `Connected as ${myClientId}`;
                    updateGameSelectionUI();
                    break;
                case 'update_lobby':
                    updatePlayerList(message.payload.players);
                    break;
                case 'game_selected':
                    selectedGameDiv.textContent = `Game selected by server: ${message.payload.selectedGame}! Waiting for game start...`;
                    // Hier könnte man die Spielauswahl-Buttons deaktivieren etc.
                    // Reset game buttons after a delay?
                    setTimeout(() => {
                         selectedGameDiv.textContent = '';
                         updateGameSelectionUI(); // Buttons wieder aktivieren
                     }, 5000); // Nach 5 Sek wieder Auswahl ermöglichen (nur Beispiel)
                    break;
                case 'error':
                     logMessage(`Server Error: ${message.payload.message}`);
                     alert(`Server Error: ${message.payload.message}`);
                     break;
                default:
                    logMessage(`Unknown message type: ${message.type}`);
            }
        }

         function updatePlayerList(players) {
            playersDiv.innerHTML = '<h2>Players</h2>'; // Clear old list
            if (players) {
                for (const [clientId, score] of Object.entries(players)) {
                    const playerDiv = document.createElement('div');
                    playerDiv.className = 'player';
                    let name = clientId;
                    if (clientId === myClientId) {
                        name += ' (You)';
                    }
                    playerDiv.innerHTML = `<strong>${name}:</strong> ${score} points`;
                    playersDiv.appendChild(playerDiv);
                }
            }
        }

        function updateGameSelectionUI() {
            gamesDiv.innerHTML = '<h2>Select a Game</h2>'; // Clear old buttons
            if (availableGames.length > 0) {
                availableGames.forEach(game => {
                    const button = document.createElement('button');
                    button.textContent = game;
                    button.onclick = () => selectGame(game);
                    gamesDiv.appendChild(button);
                });
            } else {
                gamesDiv.innerHTML += '<p>No games available currently.</p>';
            }
        }

        function selectGame(gameName) {
            if (!ws || ws.readyState !== WebSocket.OPEN) {
                logMessage('Not connected.');
                return;
            }
            const payload = { game: gameName };
            const message = {
                type: 'select_game',
                payload: payload
            };
            const messageString = JSON.stringify(message);
            logMessage(`-> Sending: ${messageString}`);
            ws.send(messageString);
            selectedGameDiv.textContent = `You selected: ${gameName}. Waiting for others...`;
            // Deaktiviere Buttons nach Auswahl
            gamesDiv.querySelectorAll('button').forEach(btn => btn.disabled = true);
        }

        // Initial connection
        connect();
    </script>
</body>
</html>
```

**Zusammenfassung der Logik:**

1.  **Verbindung:** Ein Client verbindet sich über `/ws`. Der Server erstellt ein `Client`-Objekt mit einer UUID, registriert es beim `Hub` und startet Goroutinen (`readPump`, `writePump`) für die Kommunikation.
2.  **Registrierung im Hub:** Der Hub fügt den Client seiner `clients`-Map hinzu, sendet dem neuen Client eine `welcome`-Nachricht (mit seiner ID und den Spielen) und sendet ein `update_lobby` an alle.
3.  **Nachrichten vom Client:** Der `readPump` des Clients liest Nachrichten. Wenn eine `select_game`-Nachricht kommt, wird sie über den `incoming`-Kanal an den Hub gesendet.
4.  **Verarbeitung im Hub:** Der Hub empfängt die Nachricht im `run()`-Loop. Er validiert die Spielauswahl, speichert sie in `currentGameSelections` und ruft `checkAllPlayersSelectedGame` auf.
5.  **Spielauswahl:** `checkAllPlayersSelectedGame` prüft, ob die Anzahl der Einträge in `currentGameSelections` der Anzahl der Clients entspricht. Wenn ja, ruft er `selectAndAnnounceGame` auf.
6.  **Zufällige Auswahl:** `selectAndAnnounceGame` wählt *zufällig* eines der von den Spielern ausgewählten Spiele aus (noch keine Gewichtung implementiert). Es sendet eine `game_selected`-Nachricht an alle Clients und setzt die `currentGameSelections` für die nächste Runde zurück.
7.  **Score-Update (Platzhalter):** Die Funktion `updateScores` zeigt, wie Scores später (nach einem Spiel) aktualisiert und an alle gebroadcastet werden könnten.
8.  **Trennung:** Wenn ein Client die Verbindung trennt (oder ein Fehler auftritt), beendet sich der `readPump` (oder `writePump`), sendet den Client an den `unregister`-Kanal des Hubs. Der Hub entfernt den Client, schließt seinen `send`-Kanal und sendet ein `update_lobby` an die verbleibenden Clients.

**Nächste Schritte & Verbesserungen:**

1.  **Gewichtete Zufallsauswahl:** Implementiere die Logik in `selectAndAnnounceGame`, um Spiele basierend darauf auszuwählen, wie oft sie gewählt wurden.
2.  **Tatsächliche Spiellogik:** Wenn ein Spiel ausgewählt wird (`game_selected`), muss der Server den Spielzustand initialisieren und Nachrichten für das spezifische Spiel verarbeiten. Dies könnte in separaten Modulen oder Hubs geschehen.
3.  **Score-Updates:** Nach Abschluss eines Spiels muss die Spiellogik die Ergebnisse (wer wie viele Punkte bekommt) an den Hub zurückmelden, der dann `updateScores` aufruft.
4.  **Fehlerbehandlung:** Robusteres Error-Handling hinzufügen (z.B. was passiert, wenn JSON ungültig ist?).
5.  **Sicherheit:** `CheckOrigin` im `websocket.Upgrader` korrekt konfigurieren. Ggf. Authentifizierung hinzufügen.
6.  **Persistenz:** Scores könnten in einer Datenbank gespeichert werden, um sie über Server-Neustarts hinweg zu erhalten.
7.  **Struktur:** Bei komplexeren Spielen könnte man den Hub weiter aufteilen (z.B. einen Lobby-Hub und separate Game-Hubs).
8.  **Client-Darstellung:** Im Frontend könnte man anzeigen, wer bereits gewählt hat.

Dieser Code bietet eine solide Grundlage für dein Multiplayer-Spiel mit einer zentralen Lobby und dem Spielauswahlmechanismus. Du kannst darauf aufbauen, um die spezifische Spiellogik zu integrieren.
