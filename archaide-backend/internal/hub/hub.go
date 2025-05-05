package hub

import (
	"encoding/json"
	"log"
	"math/rand"
	"slices"
	"sync"
	"time"

	"github.com/Driemtax/Archaide/internal/game"
	"github.com/Driemtax/Archaide/internal/game/asteroids"
	"github.com/Driemtax/Archaide/internal/game/pong"
	"github.com/Driemtax/Archaide/internal/message"
	"github.com/google/uuid"
)

type hubMessage struct {
	client  *Client
	message message.Message
}

type Hub struct {
	clients               map[*Client]bool
	incoming              chan hubMessage
	Register              chan *Client
	unregister            chan *Client
	availableGames        []string
	currentGameSelections map[*Client]string
	activeGames           map[string]game.Game
	clientToGame          map[*Client]string // Key: Client, Value: Game-ID
	// Always lock before writing to on of the global states!!!
	// Bad unspeakable things happened before I added this :cry:
	gameMutex sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		incoming:              make(chan hubMessage, 256),
		Register:              make(chan *Client),
		unregister:            make(chan *Client),
		clients:               make(map[*Client]bool),
		availableGames:        []string{"Asteroids", "Pong"},
		currentGameSelections: make(map[*Client]string),
		activeGames:           make(map[string]game.Game),
		clientToGame:          make(map[*Client]string),
	}
}

func (h *Hub) Run() {
	log.Println("Hub is running...")
	for {
		select {
		case client := <-h.Register:
			h.gameMutex.Lock()
			h.clients[client] = true
			h.gameMutex.Unlock()
			log.Printf("Client %s registered. Total clients: %d", client.Id, len(h.clients))

			welcomePayload := message.WelcomeMessage{
				ClientID:     client.Id,
				CurrentGames: h.availableGames,
			}
			client.SendMessage(message.Welcome, welcomePayload)
			h.broadcastLobbyUpdate()

		case client := <-h.unregister:
			h.gameMutex.Lock()
			if _, ok := h.clients[client]; ok {
				gameID, inGame := h.clientToGame[client]
				if inGame {
					if activeGame, gameExists := h.activeGames[gameID]; gameExists {
						activeGame.RemovePlayer(client)
						log.Printf("Removed client %s from game %s", client.GetID(), activeGame.GetID())
						// TODO check if the game has to be stopped and terminated
						// We should move all player back to the lobby
					}
					delete(h.clientToGame, client)
				}
				delete(h.clients, client)
				delete(h.currentGameSelections, client)
				close(client.Send)
				log.Printf("Client %s unregistered. Total clients: %d", client.Id, len(h.clients))
			}
			h.gameMutex.Unlock()
			h.broadcastLobbyUpdate()
			// Check and only start the game if all players have selected a game
			h.checkAndPotentiallyStartGame()

		case hubMsg := <-h.incoming:
			h.gameMutex.RLock()
			gameID, inGame := h.clientToGame[hubMsg.client]
			h.gameMutex.RUnlock()

			if inGame {
				h.gameMutex.RLock()
				currentGame, gameExists := h.activeGames[gameID]
				h.gameMutex.RUnlock()

				if gameExists {
					// Redirect the incoming message to the currently running game
					currentGame.HandleMessage(hubMsg.client, hubMsg.message)
				} else {
					log.Printf("Client %s mapped to game %s, but game does not exist.", hubMsg.client.GetID(), gameID)
					h.gameMutex.Lock()
					delete(h.clientToGame, hubMsg.client)
					h.gameMutex.Unlock()
				}
			} else {
				h.handleLobbyMessage(hubMsg.client, hubMsg.message)
			}
		}
	}
}

// Handles all messages from clients that are not inside a game
func (h *Hub) handleLobbyMessage(client *Client, msg message.Message) {
	switch msg.Type {
	case message.SelectGame:
		var payload message.SelectGamePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshalling select_game payload from %s: %v", client.Id, err)
			client.SendMessage(message.Error, message.ErrorMessage{Message: "Invalid select_game payload"})
			return
		}

		isValidGame := slices.Contains(h.availableGames, payload.Game)
		if !isValidGame {
			log.Printf("Client %s selected invalid game: %s", client.Id, payload.Game)
			client.SendMessage(message.Error, message.ErrorMessage{Message: "Invalid game selected"})
			return
		}

		h.gameMutex.Lock()
		h.currentGameSelections[client] = payload.Game
		client.SelectedGame = payload.Game
		log.Printf("Client %s selected game: %s", client.Id, payload.Game)
		h.gameMutex.Unlock()

		h.gameMutex.RLock()
		allSelected := h.checkAllPlayersSelectedGameInternal()
		h.gameMutex.RUnlock()

		if allSelected {
			log.Printf("All %d players have selected a game. Determining winner...", len(h.clients))
			h.selectAndStartGame()
		} else {
			log.Printf("%d out of %d players have selected a game.", len(h.currentGameSelections), len(h.clients))
		}

	default:
		log.Printf("Received unhandled lobby message type '%s' from client %s", msg.Type, client.Id)
	}
}

// checkAllPlayersSelectedGameInternal prüft intern, ob alle gewählt haben (benötigt externen Lock)
func (h *Hub) checkAllPlayersSelectedGameInternal() bool {
	if len(h.clients) == 0 {
		// We can't start a game without having clients
		return false
	}
	// Check all clients that are currently not inside of a game
	lobbyClients := 0
	selectedCount := 0
	for client := range h.clients {
		if _, inGame := h.clientToGame[client]; !inGame {
			lobbyClients++
			if _, selected := h.currentGameSelections[client]; selected {
				selectedCount++
			}
		}
	}

	// Only starts if at least two players are inside of the lobby
	// and every player has selected a game
	return lobbyClients > 0 && selectedCount == lobbyClients
}

// Selects a game from the player selections, creates a new instance
// of the game and starts it
func (h *Hub) selectAndStartGame() {
	h.gameMutex.Lock()

	if len(h.currentGameSelections) == 0 {
		log.Println("No selections made, cannot select a game.")
		return
	}

	selections := []string{}
	participatingClients := []*Client{} // All the clients that will join the new game
	for client, gameName := range h.currentGameSelections {
		// Important late night note:
		// Only add players to a game that are not inside a game yet *in anger of my own stupidity*
		if _, inGame := h.clientToGame[client]; !inGame {
			selections = append(selections, gameName)
			participatingClients = append(participatingClients, client)
		}
	}

	if len(participatingClients) == 0 {
		log.Println("All selecting clients are already in games? Cannot start.")
		// Reset selections for safety
		h.currentGameSelections = make(map[*Client]string)
		for client := range h.clients {
			client.SelectedGame = ""
		}
		return
	}

	// Selects a game and also takes the amount of votes into account
	// because selections has all the selections...
	randomIndex := rand.Intn(len(selections))
	selectedGameName := selections[randomIndex]

	log.Printf("Selected game: %s for %d players", selectedGameName, len(participatingClients))

	/// --- Creating the new game instance ---
	var newGame game.Game
	gameID := uuid.New().String()

	switch selectedGameName {
	case "Asteroids":
		asteroidsGame := asteroids.NewAsteroidsGame(h, gameID)
		newGame = asteroidsGame
		log.Printf("Instantiated Asteroids game with ID %s", gameID)

	case "Pong":
		pongGame := pong.NewPongGame(h, gameID)
		newGame = pongGame
		log.Printf("Instantiated Pong game with ID %s", gameID)

	default:
		log.Printf("Unknown game selected: %s", selectedGameName)
		return
	}

	// Register game and clients
	h.activeGames[gameID] = newGame
	for _, client := range participatingClients {
		h.clientToGame[client] = gameID
		err := newGame.AddPlayer(client)
		if err != nil {
			log.Printf("Error adding player %s to game %s: %v", client.Id, gameID, err)
			// TODO error handling
			// Should we stop the game or smth else? Im not sure yet
			// Currently the player just wont get added to the game
			delete(h.clientToGame, client)
		} else {
			// Inform the client that a game will start
			startPayload := message.GameSelectedMessage{SelectedGame: selectedGameName, GameID: gameID}
			client.SendMessage(message.GameSelected, startPayload)
			log.Printf("Added player %s to game %s", client.Id, gameID)
		}
	}

	// Start the game in a new goroutine
	go newGame.Start()
	log.Printf("Started game %s (%s) in a new goroutine", gameID, selectedGameName)

	// Lets clear all previous game selections
	for _, client := range participatingClients {
		delete(h.currentGameSelections, client)
		client.SelectedGame = ""
	}

	log.Printf("Cleared all previous game selection!\n")

	// Please unlock mutex here, scince broadcastLobbyUpdate also tries to Lock.
	// It was a very painful sunday morning :cry:
	h.gameMutex.Unlock()
	// Broadcast to all players the new Lobby state
	h.broadcastLobbyUpdate()
}

// Has to be called from a game after it is finished
func (h *Hub) GameFinished(gameID string, result game.GameResult) {
	h.gameMutex.Lock()
	defer h.gameMutex.Unlock()

	log.Printf("Game %s finished. Processing results.", gameID)

	// Remove the game from the current active games!
	if _, exists := h.activeGames[gameID]; exists {
		delete(h.activeGames, gameID)
	} else {
		// If the game has already been finished for some reason...
		// We just quit the function here :)
		log.Printf("GameFinished called for non-existent or already finished game %s", gameID)
		return
	}

	// Remove clients from the client to game mapping
	clientsToRemove := []*Client{}
	for client, gid := range h.clientToGame {
		if gid == gameID {
			clientsToRemove = append(clientsToRemove, client)
		}
	}
	for _, client := range clientsToRemove {
		delete(h.clientToGame, client)
		client.gameID = ""                           // the client is back in the lobby
		client.SendMessage(message.BackToLobby, nil) // notify the client that hes back in the lobby!
		log.Printf("Client %s removed from finished game %s, returned to lobby.", client.GetID(), gameID)
	}

	// Update all scores if scores have been given
	if result.Scores != nil && len(result.Scores) > 0 {
		h.updateScoresInternal(result.Scores)
	}

	// Notify all players for the lobby update
	h.broadcastLobbyUpdate()

	// At this point it will again be checked if a new game can be started...
	// Using time.AfterFunc for a small delay, gives clients time to process
	// Im not completly sure that this here is the best way to do it, but it
	// works fine for now so i will come back to it if it creates some problems
	time.AfterFunc(500*time.Millisecond, h.checkAndPotentiallyStartGame)
}

func (h *Hub) broadcastLobbyUpdate() {
	playerInfos := make(map[string]message.PlayerInfo)
	h.gameMutex.RLock()
	for client := range h.clients {
		// Check if the client is currently inside a game
		_, inGame := h.clientToGame[client]
		playerInfos[client.Id] = message.PlayerInfo{
			Score:  client.Score,
			InGame: inGame,
		}
	}
	h.gameMutex.RUnlock()
	payload := message.LobbyUpdateMessage{Players: playerInfos}

	h.broadcastMessageInternal(message.UpdateLobby, payload)
}

// BroadcastMessage - Sendet an ALLE verbundenen Clients (wird jetzt intern genutzt)
func (h *Hub) broadcastMessageInternal(msgType message.MessageType, payload any) {
	h.gameMutex.RLock()
	log.Printf("Broadcasting message type '%s' to %d clients", msgType, len(h.clients))
	clientList := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clientList = append(clientList, client)
	}
	h.gameMutex.RUnlock()

	for _, client := range clientList {
		err := client.SendMessage(msgType, payload)
		if err != nil {
			log.Printf("Error broadcasting message type %s to client %s: %v", msgType, client.Id, err)
		}
	}
}

// Checks if possible and starts a game
func (h *Hub) checkAndPotentiallyStartGame() {
	h.gameMutex.RLock()
	allSelected := h.checkAllPlayersSelectedGameInternal()
	// At least two players have to be there
	canStart := len(h.clients) > 1 && allSelected
	h.gameMutex.RUnlock()

	if canStart {
		log.Printf("All %d lobby players have selected a game. Determining winner...", len(h.currentGameSelections)) // Logik hier anpassen
		h.selectAndStartGame()
	} else {
		h.gameMutex.RLock()
		lobbyClientsCount := 0
		for c := range h.clients {
			if _, inGame := h.clientToGame[c]; !inGame {
				lobbyClientsCount++
			}
		}
		selectedCount := len(h.currentGameSelections)
		h.gameMutex.RUnlock()
		if lobbyClientsCount > 0 {
			log.Printf("%d out of %d lobby players have selected a game.", selectedCount, lobbyClientsCount)
		}
	}
}

// Helper function to reset all the selections
func (h *Hub) resetSelections(clients []*Client) {
	for _, client := range clients {
		delete(h.currentGameSelections, client)
		client.SelectedGame = ""
	}
}

func (h *Hub) updateScoresInternal(scores map[string]int) {
	log.Println("Updating scores...")
	for clientID, delta := range scores {
		var targetClient *Client = nil
		for c := range h.clients {
			if c.GetID() == clientID {
				targetClient = c
				break
			}
		}
		if targetClient != nil {
			targetClient.Score += delta
			log.Printf("Score updated for %s: new score %d", targetClient.GetID(), targetClient.Score)
		} else {
			log.Printf("Could not find client %s to update score", clientID)
		}
	}
}

// An api that the hub implements which can be passed
// down to the game
type GameFinisher interface {
	GameFinished(gameID string, result game.GameResult)
}

// Checking if the hub implements the game finished interface correctly
var _ GameFinisher = (*Hub)(nil)
