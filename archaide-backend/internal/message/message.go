package message

import (
	"encoding/json"
)

// Message represents a generic message that is sent over WebSocket.
// The 'Type' helps the server or client understand how to interpret the 'Payload'.
type Message struct {
	Type    MessageType     `json:"type"`    // e.g. "update_lobby", "select_game", "error", "welcome"
	Payload json.RawMessage `json:"payload"` // The actual data, depending on the type
}

type MessageType string

const (
	// Message types for the WebSocket communication
	Welcome           MessageType = "welcome"             // Sent when a client connects
	BackToLobby       MessageType = "back_to_lobby"       // Send when a player returns from a game back to the lobby
	UpdateLobby       MessageType = "update_lobby"        // Sent to update the lobby state
	SelectGame        MessageType = "select_game"         // Sent when a client selects a game
	GameSelected      MessageType = "game_selected"       // Sent when a game is selected
	Error             MessageType = "error"               // Sent when an error occurs
	PongInput         MessageType = "pong_input"          // From client: Move paddle
	PongState         MessageType = "pong_state"          // From server: current game state
	PongGameOver      MessageType = "pong_game_over"      // From server: game over
	AsteroidsInput    MessageType = "asteroids_input"     // From client: Move player
	AsteroidsState    MessageType = "asteroids_state"     // From server: current game state
	AsteroidsGameOver MessageType = "asteroids_game_over" // From server: game over
)

// Payload Structures (Examples)

// WelcomeMessage contains the ID of the new client and the list of available games
type WelcomeMessage struct {
	ClientID     string   `json:"clientId"`
	CurrentGames []string `json:"currentGames"`
}

type PlayerInfo struct {
	Score  int  `json:"score"`
	InGame bool `json:"inGame"`
}

// LobbyUpdateMessage contains the current state of the lobby (players and their scores)
type LobbyUpdateMessage struct {
	Players map[string]PlayerInfo `json:"players"` // Map of ClientID to Score
}

// SelectGamePayload is sent by the client when they select a game
type SelectGamePayload struct {
	Game string `json:"game"`
}

// GameSelectedMessage is sent to all when a game is selected
type GameSelectedMessage struct {
	SelectedGame string `json:"selectedGame"`
	GameID       string `json:"gameId"`
}

// ErrorMessage is sent in case of errors
type ErrorMessage struct {
	Message string `json:"message"`
}

/// --- PONG ---

type PongInputPayload struct {
	Direction string `json:"direction"` // "up", "down", "none"
}
type PongStatePayload struct {
	BallX    float64 `json:"ballX"`
	BallY    float64 `json:"ballY"`
	Paddle1Y float64 `json:"paddle1Y"`
	Paddle2Y float64 `json:"paddle2Y"`
	Score1   int     `json:"score1"`
	Score2   int     `json:"score2"`
}

type PongGameOverPayload struct {
	Winner string `json:"winner"` // ClientID of the winner
}
