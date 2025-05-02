package games

import (
	"github.com/Driemtax/Archaide/internal/coms"
)

// Game Interface represents a game that can be played in the lobby and must be implemented by all games
type Game interface {
	Run()                                          // Starts the game
	HandleInput(client *coms.Client, input []byte) // Handles input from a client
	isOver() bool                                  // Checks if the game is over
	sendGameState()                                // Sends the current game state to all clients
	sendGameOver()                                 // Sends the game over state to all clients
}
