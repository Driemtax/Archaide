package games

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Driemtax/Archaide/internal/component"
	"github.com/Driemtax/Archaide/internal/coms"
	"github.com/Driemtax/Archaide/internal/message"
)

/// TODO rebuild this game for multiple users

type PlayerInputMovement string

const (
	North        PlayerInputMovement = "north"
	East         PlayerInputMovement = "east"
	South        PlayerInputMovement = "south"
	West         PlayerInputMovement = "west"
	NorthEast    PlayerInputMovement = "north_east"
	NorthWest    PlayerInputMovement = "north_west"
	SouthWest    PlayerInputMovement = "south_west"
	SouthEast    PlayerInputMovement = "south_east"
	PLAYER_SPEED                     = 5
)

type AsteroidsGame struct {
	Client1, Client2 *coms.Client
	Client1Input     chan []byte
	Client2Input     chan []byte

	Player1 *AsteroidsPlayer
	Player2 *AsteroidsPlayer

	isRunning bool
}

type AsteroidsPlayer struct {
	Pos   component.Vector2D `json:"pos"`
	Speed float64            `json:"speed"`
	Dir   component.Vector2D `json:"dir"`
}

func NewAsteroidsPlayer(x, y float64) *AsteroidsPlayer {
	return &AsteroidsPlayer{
		Pos:   component.NewVector2D(x, y),
		Speed: PLAYER_SPEED,
		Dir:   component.NewVector2D(0, -1),
	}
}

func (ap *AsteroidsPlayer) UpdateDir(mov PlayerInputMovement) {
	var dir component.Vector2D
	switch mov {
	case North:
		dir.Y -= 1
	case East:
		dir.X += 1
	case South:
		dir.Y += 1
	case West:
		dir.X -= 1
	case NorthEast:
		dir.X += 1
		dir.Y -= 1
	case NorthWest:
		dir.X -= 1
		dir.Y -= 1
	case SouthEast:
		dir.X += 1
		dir.Y += 1
	case SouthWest:
		dir.X -= 1
		dir.Y += 1
	}

	dir.Normalize()
	ap.Dir = dir
}

func NewAsteroidsGame(c1, c2 *coms.Client) *AsteroidsGame {
	return &AsteroidsGame{
		Client1:      c1,
		Client2:      c2,
		Client1Input: make(chan []byte),
		Client2Input: make(chan []byte),

		Player1: NewAsteroidsPlayer(0, 0),
		Player2: NewAsteroidsPlayer(0, 0),

		isRunning: false,
	}
}

func (g *AsteroidsGame) Run() {
	g.isRunning = true
	ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
	defer ticker.Stop()

	for g.isRunning {
		select {
		case input := <-g.Client1Input:
			g.HandleInput(g.Client1, input)
		case input := <-g.Client2Input:
			g.HandleInput(g.Client2, input)
		case <-ticker.C:
			g.Update()
			g.sendGameState()
			if g.isOver() {
				g.isRunning = false
				g.sendGameOver()
				return
			}
		}
	}
}

func (g *AsteroidsGame) HandleInput(client *coms.Client, input []byte) {
	var msg AsteroidsInputPayload
	if err := json.Unmarshal(input, &msg); err != nil {
		fmt.Println("Error unmarshaling input in Asteroids game:", err)
		return
	}

	if client.Id == g.Client1.Id {
		g.Player1.UpdateDir(msg.Direction)
	} else if client.Id == g.Client2.Id {
		g.Player2.UpdateDir(msg.Direction)
	}
}

func (g *AsteroidsGame) Update() {
	// Update all the asteroids
	// Update player 1
	g.Player1.Pos = g.Player1.Dir.Mul(PLAYER_SPEED)
	// Update player 2
	g.Player2.Pos = g.Player2.Dir.Mul(PLAYER_SPEED)
}

func (g *AsteroidsGame) isOver() bool {
	return false
}

func (g *AsteroidsGame) sendGameState() {
	gameState := AsteroidsStatePayload{
		Player1Pos: g.Player1.Pos,
		Player2Pos: g.Player2.Pos,
	}

	g.Client1.SendMessage(message.AsteroidsState, gameState)
	g.Client2.SendMessage(message.AsteroidsState, gameState)
}

func (g *AsteroidsGame) sendGameOver() {
	gameOverMessage := "The game is over what so ever"
	g.Client1.SendMessage(message.AsteroidsGameOver, gameOverMessage)
	g.Client2.SendMessage(message.AsteroidsGameOver, gameOverMessage)
}

/// Messages

type AsteroidsInputPayload struct {
	Direction PlayerInputMovement `json:"direction"`
}

type AsteroidsStatePayload struct {
	Player1Pos component.Vector2D `json:"player1"`
	Player2Pos component.Vector2D `json:"player2"`
}

type AsteroidsGameOverPayload struct {
	Winner string `json:"winner"`
}
