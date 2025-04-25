package coms

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/Driemtax/Archaide/internal/message"
)

// HubMessage ist eine Wrapper-Struktur, um Nachrichten zusammen mit dem sendenden Client an den Hub zu übergeben.
type HubMessage struct {
	client  *Client
	message message.Message
}

// Hub verwaltet den Satz aktiver Clients und broadcastet Nachrichten an sie.
type Hub struct {
	// Registrierte Clients. Die Keys sind die Client-Pointer, der Wert ist immer true.
	// Oder: map[string]*Client für einfacheren Zugriff per ID
	Clients map[*Client]bool

	// Eingehende Nachrichten von den Clients.
	Incoming chan HubMessage

	// Registrierungsanfragen von Clients.
	Register chan *Client

	// Deregistrierungsanfragen von Clients.
	Unregister chan *Client

	// Liste der verfügbaren Spiele
	AvailableGames []string

	// Spielauswahlen der aktuellen Runde (Client -> Spielname)
	CurrentGameSelections map[*Client]string
}

func NewHub() *Hub {
	return &Hub{
		Incoming:              make(chan HubMessage),
		Register:              make(chan *Client),
		Unregister:            make(chan *Client),
		Clients:               make(map[*Client]bool),
		AvailableGames:        []string{"Asteroids", "Pong", "Space Invaders"}, // Beispielspiele
		CurrentGameSelections: make(map[*Client]string),
	}
}

func (h *Hub) Run() {
	log.Println("Hub is running...")
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			log.Printf("Client %s registered. Total clients: %d", client.Id, len(h.Clients))

			// Sende eine Willkommensnachricht an den neuen Client
			welcomePayload := message.WelcomeMessage{
				ClientID:     client.Id,
				CurrentGames: h.AvailableGames,
			}
			client.sendMessage("welcome", welcomePayload)

			// Sende den aktuellen Lobby-Status an alle Clients
			h.broadcastLobbyUpdate()

		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				delete(h.CurrentGameSelections, client) // Entferne auch die Auswahl
				close(client.Send)                      // Schließe den Send-Kanal des Clients
				log.Printf("Client %s unregistered. Total clients: %d", client.Id, len(h.Clients))
				// Sende den aktualisierten Lobby-Status an die verbleibenden Clients
				h.broadcastLobbyUpdate()
				// Überprüfe nach dem Verlassen, ob nun alle gewählt haben (falls jemand geht, während abgestimmt wird)
				h.checkAllPlayersSelectedGame()
			}

		case hubMsg := <-h.Incoming:
			// Verarbeite die Nachricht vom Client
			h.handleIncomingMessage(hubMsg.client, hubMsg.message)
		}
	}
}

// Verarbeitet eingehende Nachrichten von einem Client
func (h *Hub) handleIncomingMessage(client *Client, msg message.Message) {
	log.Printf("Received message type '%s' from client %s", msg.Type, client.Id)
	switch msg.Type {
	case "select_game":
		var payload message.SelectGamePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshalling select_game payload from %s: %v", client.Id, err)
			client.sendMessage("error", message.ErrorMessage{Message: "Invalid select_game payload"})
			return
		}

		// Validieren, ob das Spiel gültig ist
		isValidGame := false
		for _, game := range h.AvailableGames {
			if game == payload.Game {
				isValidGame = true
				break
			}
		}
		if !isValidGame {
			log.Printf("Client %s selected invalid game: %s", client.Id, payload.Game)
			client.sendMessage("error", message.ErrorMessage{Message: "Invalid game selected"})
			return
		}

		// Speichere die Auswahl des Spielers
		h.CurrentGameSelections[client] = payload.Game
		client.SelectedGame = payload.Game // Auch im Client speichern
		log.Printf("Client %s selected game: %s", client.Id, payload.Game)

		// Sende ggf. eine Bestätigung oder Update an Clients (optional)
		// client.sendMessage("game_selection_received", ...)

		// Überprüfe, ob alle Spieler gewählt haben
		h.checkAllPlayersSelectedGame()

	// Hier könnten weitere Nachrichten-Typen behandelt werden (z.B. Chat)
	default:
		log.Printf("Received unhandled message type '%s' from client %s", msg.Type, client.Id)
	}
}

// Überprüft, ob alle verbundenen Spieler ein Spiel für die aktuelle Runde ausgewählt haben
func (h *Hub) checkAllPlayersSelectedGame() {
	if len(h.Clients) == 0 {
		return // Niemand da, nichts zu tun
	}

	allSelected := true
	for client := range h.Clients {
		if _, ok := h.CurrentGameSelections[client]; !ok {
			allSelected = false
			break
		}
	}

	if allSelected {
		log.Printf("All %d players have selected a game. Determining winner...", len(h.Clients))
		h.selectAndAnnounceGame()
		// Setze die Auswahlen für die nächste Runde zurück
		h.CurrentGameSelections = make(map[*Client]string)
		for client := range h.Clients {
			client.SelectedGame = "" // Auch im Client zurücksetzen
		}
	} else {
		log.Printf("%d out of %d players have selected a game.", len(h.CurrentGameSelections), len(h.Clients))
		// Optional: Sende ein Update, wer noch nicht gewählt hat
	}
}

// Wählt zufällig ein Spiel basierend auf den Auswahlen aus und kündigt es an
func (h *Hub) selectAndAnnounceGame() {
	if len(h.CurrentGameSelections) == 0 {
		log.Println("No selections made, cannot select a game.")
		return
	}

	// Einfache zufällige Auswahl aus den gewählten Spielen
	// TODO: Implementiere die gewichtete Auswahl basierend auf der Häufigkeit der Auswahl
	selections := []string{}
	for _, gameName := range h.CurrentGameSelections {
		selections = append(selections, gameName)
	}

	rand.Seed(time.Now().UnixNano()) // Seed für Zufallszahlengenerator
	randomIndex := rand.Intn(len(selections))
	selectedGame := selections[randomIndex]

	log.Printf("Randomly selected game: %s", selectedGame)

	// Sende das Ergebnis an alle Clients
	announcementPayload := message.GameSelectedMessage{SelectedGame: selectedGame}
	h.broadcastMessage("game_selected", announcementPayload)

	// --- Hier würde die Logik zum Starten des Spiels beginnen ---
	// Zum Beispiel: Sende eine "start_game" Nachricht mit Details
	// h.startGame(selectedGame)
}

// broadcastLobbyUpdate sendet den aktuellen Spielerstatus (ID und Score) an alle Clients.
func (h *Hub) broadcastLobbyUpdate() {
	playerScores := make(map[string]int)
	for client := range h.Clients {
		playerScores[client.Id] = client.Score
	}
	payload := message.LobbyUpdateMessage{Players: playerScores}

	h.broadcastMessage("update_lobby", payload)
}

// broadcastMessage sendet eine Nachricht an alle verbundenen Clients.
func (h *Hub) broadcastMessage(msgType string, payload interface{}) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling payload for broadcast: %v", err)
		return
	}
	message := message.Message{
		Type:    msgType,
		Payload: json.RawMessage(payloadBytes),
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshalling message for broadcast: %v", err)
		return
	}

	log.Printf("Broadcasting message type '%s' to %d clients", msgType, len(h.Clients))
	for client := range h.Clients {
		select {
		case client.Send <- messageBytes:
		default:
			// Wenn der Send-Kanal blockiert oder geschlossen ist,
			// gehen wir davon aus, dass der Client langsam oder getrennt ist.
			// Wir entfernen ihn hier nicht direkt, das sollte der readPump/writePump tun.
			log.Printf("Could not send broadcast to client %s (send buffer full or closed)", client.Id)
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
		for c := range h.Clients {
			if c.Id == clientID {
				targetClient = c
				break
			}
		}

		if targetClient != nil {
			targetClient.Score += delta
			log.Printf("Score updated for %s: new score %d", targetClient.Id, targetClient.Score)
		} else {
			log.Printf("Could not find client %s to update score", clientID)
		}
	}
	// Sende die neuen Scores an alle
	h.broadcastLobbyUpdate()
}
