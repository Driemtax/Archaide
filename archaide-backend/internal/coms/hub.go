package coms

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/Driemtax/Archaide/internal/message"
)

// HubMessage wraps a message with its sending client for hub processing
type HubMessage struct {
	client  *Client
	message message.Message
}

// Hub manages active clients and broadcasts messages between them
type Hub struct {
	Clients               map[*Client]bool
	Incoming              chan HubMessage
	Register              chan *Client
	Unregister            chan *Client
	AvailableGames        []string
	CurrentGameSelections map[*Client]string
}

func NewHub() *Hub {
	return &Hub{
		Incoming:              make(chan HubMessage),
		Register:              make(chan *Client),
		Unregister:            make(chan *Client),
		Clients:               make(map[*Client]bool),
		AvailableGames:        []string{"Asteroids", "Pong", "Space Invaders"},
		CurrentGameSelections: make(map[*Client]string),
	}
}

// Run starts the hub's main event loop to handle client connections and messages
func (h *Hub) Run() {
	log.Println("Hub is running...")
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			log.Printf("Client %s registered. Total clients: %d", client.Id, len(h.Clients))

			welcomePayload := message.WelcomeMessage{
				ClientID:     client.Id,
				CurrentGames: h.AvailableGames,
			}
			client.SendMessage(message.Welcome, welcomePayload)

			h.broadcastLobbyUpdate()

		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				delete(h.CurrentGameSelections, client)
				close(client.Send)
				log.Printf("Client %s unregistered. Total clients: %d", client.Id, len(h.Clients))
				h.broadcastLobbyUpdate()
				h.checkAllPlayersSelectedGame()
			}

		case hubMsg := <-h.Incoming:
			h.handleIncomingMessage(hubMsg.client, hubMsg.message)
		}
	}
}

// handleIncomingMessage processes messages received from clients based on their type
func (h *Hub) handleIncomingMessage(client *Client, msg message.Message) {
	log.Printf("Received message type '%s' from client %s", msg.Type, client.Id)
	switch msg.Type {
	case message.SelectGame:
		var payload message.SelectGamePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshalling select_game payload from %s: %v", client.Id, err)
			client.SendMessage("error", message.ErrorMessage{Message: "Invalid select_game payload"})
			return
		}

		isValidGame := false
		for _, game := range h.AvailableGames {
			if game == payload.Game {
				isValidGame = true
				break
			}
		}
		if !isValidGame {
			log.Printf("Client %s selected invalid game: %s", client.Id, payload.Game)
			client.SendMessage("error", message.ErrorMessage{Message: "Invalid game selected"})
			return
		}

		h.CurrentGameSelections[client] = payload.Game
		client.SelectedGame = payload.Game
		log.Printf("Client %s selected game: %s", client.Id, payload.Game)

		h.checkAllPlayersSelectedGame()

	default:
		log.Printf("Received unhandled message type '%s' from client %s", msg.Type, client.Id)
	}
}

// checkAllPlayersSelectedGame determines if all connected players have made a game selection
// and triggers the game selection process if everyone has voted
func (h *Hub) checkAllPlayersSelectedGame() {
	if len(h.Clients) == 0 {
		return
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
		h.CurrentGameSelections = make(map[*Client]string)
		for client := range h.Clients {
			client.SelectedGame = ""
		}
	} else {
		log.Printf("%d out of %d players have selected a game.", len(h.CurrentGameSelections), len(h.Clients))
	}
}

// selectAndAnnounceGame randomly selects a game from player votes and broadcasts
// the selection to all clients to start the game
func (h *Hub) selectAndAnnounceGame() {
	if len(h.CurrentGameSelections) == 0 {
		log.Println("No selections made, cannot select a game.")
		return
	}

	selections := []string{}
	for _, gameName := range h.CurrentGameSelections {
		selections = append(selections, gameName)
	}

	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(selections))
	selectedGame := selections[randomIndex]

	log.Printf("Randomly selected game: %s", selectedGame)

	announcementPayload := message.GameSelectedMessage{SelectedGame: selectedGame}
	h.broadcastMessage(message.GameSelected, announcementPayload)
}

// broadcastLobbyUpdate Sends the current player list with scores to all connected clients
func (h *Hub) broadcastLobbyUpdate() {
	playerScores := make(map[string]int)
	for client := range h.Clients {
		playerScores[client.Id] = client.Score
	}
	payload := message.LobbyUpdateMessage{Players: playerScores}

	h.broadcastMessage(message.UpdateLobby, payload)
}

// broadcastMessage marshals and Sends a message to all connected clients
func (h *Hub) broadcastMessage(msgType message.MessageType, payload interface{}) {
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
			log.Printf("Could not send broadcast to client %s (send buffer full or closed)", client.Id)
		}
	}
}

// updateScores updates player scores based on game results and broadcasts the changes
func (h *Hub) updateScores(scores map[string]int) {
	log.Println("Updating scores...")
	for clientID, delta := range scores {
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
	h.broadcastLobbyUpdate()
}
