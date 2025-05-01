package message

import "encoding/json"

// Message represents a generic message that is sent over WebSocket.
// The 'Type' helps the server or client understand how to interpret the 'Payload'.
type Message struct {
	Type    string          `json:"type"`    // e.g. "update_lobby", "select_game", "error", "welcome"
	Payload json.RawMessage `json:"payload"` // The actual data, depending on the type
}

// Payload Structures (Examples)

// WelcomeMessage contains the ID of the new client and the list of available games
type WelcomeMessage struct {
	ClientID     string   `json:"clientId"`
	CurrentGames []string `json:"currentGames"`
}

// LobbyUpdateMessage contains the current state of the lobby (players and their scores)
type LobbyUpdateMessage struct {
	Players map[string]int `json:"players"` // Map of ClientID to Score
}

// SelectGamePayload is sent by the client when they select a game
type SelectGamePayload struct {
	Game string `json:"game"`
}

// GameSelectedMessage is sent to all when a game is selected
type GameSelectedMessage struct {
	SelectedGame string `json:"selectedGame"`
}

// ErrorMessage is sent in case of errors
type ErrorMessage struct {
	Message string `json:"message"`
}
