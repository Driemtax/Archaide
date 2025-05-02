package games

import (
	"github.com/Driemtax/Archaide/internal/coms"
)

// Game Interface represents a game that can be played in the lobby and must be implemented by all games
type Game interface {
	Run()                                          // Starts the game
	HandleInput(client *coms.Client, input []byte) // Handles input from a client
	AddPlayer(client *coms.Client)                 // Adds a new player to the game
	RemoverPlayer(client *coms.Client)             // Removes a player from a game
	Stop()                                         // Stops the game
}
