package games

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/Driemtax/Archaide/internal/coms"
	"github.com/Driemtax/Archaide/internal/message"
)

// Every calculation in this file depends on y=0 beeing at the bottom. I read online, that a common
// standard is to have y=0 at the top, in this case i would have to flip everything here or invert
// all y values before sending them to the client.
type PongGame struct {
	Player1, Player2           *coms.Client
	Player1Input               chan []byte
	Player2Input               chan []byte
	Player1Ready, Player2Ready bool

	BallX, BallY       float64
	BallVX, BallVY     float64
	Paddle1Y, Paddle2Y float64
	Score1, Score2     int

	Width, Height float64
	Running       bool
}

func NewPongGame(p1, p2 *coms.Client) *PongGame {
	return &PongGame{
		Player1:      p1,
		Player2:      p2,
		Player1Input: make(chan []byte, 8),
		Player2Input: make(chan []byte, 8),
		BallX:        400, BallY: 300,
		BallVX: 4, BallVY: 3,
		Paddle1Y: 250, Paddle2Y: 250,
		Width: 800, Height: 600,
		Running: true,
	}
}

const (
	PaddleWidth  = 10.0
	PaddleHeight = 50.0
	BallSize     = 10.0
)

// Runs the game loop, handling input and updating game state
func (pg *PongGame) Run() {
	// Initialisiere das Spiel
	ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
	defer ticker.Stop()
	for pg.Running {
		select {
		case input := <-pg.Player1Input:
			pg.HandleInput(pg.Player1, input)
		case input := <-pg.Player2Input:
			pg.HandleInput(pg.Player2, input)
		case <-ticker.C:
			pg.Tick()
			pg.sendGameState()
			if pg.IsOver() {
				pg.Running = false
				pg.SendGameOver()
				// Spiel beenden, Siegerehrung, Rückkehr zur Lobby
				return
			}
		}
	}
}

// HandleInput processes input from the players and moves the paddles accordingly
func (pg *PongGame) HandleInput(client *coms.Client, input []byte) {
	var inp message.PongInputPayload
	if err := json.Unmarshal(input, &inp); err != nil {
		return
	}

	speed := 8.0
	if client == pg.Player1 {
		if inp.Direction == "up" && pg.Paddle1Y < pg.Height-PaddleHeight {
			pg.Paddle1Y += speed
		} else if inp.Direction == "down" && pg.Paddle1Y > PaddleHeight {
			pg.Paddle1Y -= speed
		}
	} else if client == pg.Player2 {
		if inp.Direction == "up" && pg.Paddle2Y < pg.Height-PaddleHeight {
			pg.Paddle2Y += speed
		} else if inp.Direction == "down" && pg.Paddle2Y > PaddleHeight {
			pg.Paddle2Y -= speed
		}
	}
}

// Tick updates the game state, moving the ball, checking for collisions and scoring
func (pg *PongGame) Tick() {
	// move the ball
	pg.BallX += pg.BallVX
	pg.BallY += pg.BallVY

	// If the ball hits the top or bottom wall, reverse its Y velocity
	if pg.BallY < 0 || pg.BallY > pg.Height {
		pg.BallVY = -pg.BallVY
	}

	// TODO : Check if the ball hits the paddles
	// Check for collision with left paddle (Player 1)
	if pg.BallX <= PaddleWidth && // Ball reached the left side
		pg.BallY+BallSize >= pg.Paddle1Y && // Ball is within the paddle's height
		pg.BallY <= pg.Paddle1Y+PaddleHeight {
		pg.BallVX = -pg.BallVX // Reverse ball's X velocity
		return
	}

	// Check for collision with right paddle (Player 2)
	if pg.BallX+BallSize >= pg.Width-PaddleWidth && // Ball reached the right side
		pg.BallY+BallSize >= pg.Paddle2Y && // Ball is within the paddle's height
		pg.BallY <= pg.Paddle2Y+PaddleHeight {
		pg.BallVX = -pg.BallVX // Reverse ball's X velocity
		return
	}

	// Check if the ball hits the left or right wall, then someone scored
	if pg.BallX < 0 {
		// Player 1 scored
		pg.Score1++
		pg.Reset()
		return
	} else if pg.BallX > pg.Width {
		// Player 2 scored
		pg.Score2++
		pg.Reset()
		return

	}
}

// IsOver checks if the game is over (e.g., if a player has reached a certain score)
func (pg *PongGame) IsOver() bool {
	return pg.Score1 >= 5 || pg.Score2 >= 5
}

// sendGameState sends the current game state to both players
func (pg *PongGame) sendGameState() {
	state := message.PongStatePayload{
		BallX:    pg.BallX,
		BallY:    pg.BallY,
		Paddle1Y: pg.Paddle1Y,
		Paddle2Y: pg.Paddle2Y,
		Score1:   pg.Score1,
		Score2:   pg.Score2,
	}
	// Sende den aktuellen Spielstatus an beide Spieler
	pg.Player1.SendMessage(message.PongState, state)
	pg.Player2.SendMessage(message.PongState, state)
}

// SendGameOver sends a game over message to both players
// and determines the winner based on the scores
func (pg *PongGame) SendGameOver() {
	// Sende eine Nachricht an beide Spieler, dass das Spiel vorbei ist
	gameOverMessage := message.PongGameOverPayload{
		Winner: pg.determineWinner(),
	}
	pg.Player1.SendMessage(message.PongGameOver, gameOverMessage)
	pg.Player2.SendMessage(message.PongGameOver, gameOverMessage)
}

// determineWinner determines the winner based on the scores
func (pg *PongGame) determineWinner() string {
	if pg.Score1 > pg.Score2 {
		return pg.Player1.Id
	} else if pg.Score2 > pg.Score1 {
		return pg.Player2.Id
	}
	return "" // Draw, but is this even possible?
}

// Reset resets the game state to the initial values
func (pg *PongGame) Reset() {
	pg.BallX, pg.BallY = pg.Width/2, pg.Height/2
	vx := 4.0
	if rand.Intn(2) == 0 {
		vx = -vx
	}
	vy := 3.0
	if rand.Intn(2) == 0 {
		vy = -vy
	}
	pg.BallVX, pg.BallVY = vx, vy
	pg.Paddle1Y, pg.Paddle2Y = pg.Height/2-PaddleHeight/2, pg.Height/2-PaddleHeight/2
}
