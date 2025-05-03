package pong

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/Driemtax/Archaide/internal/game"
	"github.com/Driemtax/Archaide/internal/message"
)

const (
	// Game dimensions and elements sizes.
	// Assuming y=0 is at the bottom for these calculations. Client might need inversion.
	GameWidth    = 800.0
	GameHeight   = 600.0
	PaddleWidth  = 10.0
	PaddleHeight = 60.0 // Increased paddle height slightly
	BallSize     = 10.0

	// Game rules and physics.
	PaddleSpeed   = 10.0 // Pixels per update tick for paddle movement
	InitialBallVX = 5.0  // Initial horizontal ball speed
	InitialBallVY = 4.0  // Initial vertical ball speed
	MaxBallSpeedX = 15.0 // Prevent ball from becoming too fast horizontally
	MaxBallSpeedY = 12.0 // Prevent ball from becoming too fast vertically
	SpeedIncrease = 1.05 // Factor to increase ball speed on paddle hit
	TargetScore   = 5    // Score needed to win the game
	MinPlayers    = 2    // Required number of players
	MaxPlayers    = 2    // Maximum number of players
)

// PongPlayerState holds the game-specific state for a player in Pong.
type PongPlayerState struct {
	PlayerID string  // ID linking back to the game.Player
	PaddleY  float64 // Vertical position of the center of the paddle
	Score    int
	Role     int // 1 for Player 1 (left), 2 for Player 2 (right)
}

// PongGame implements the game.Game interface for a 2-player Pong match.
type PongGame struct {
	gameFinisher game.GameFinisher // Interface to notify the hub when the game ends
	gameID       string

	players   map[string]*PongPlayerState // Map PlayerID to their state
	playerMap map[string]game.Player      // Map PlayerID back to the Player interface for sending messages
	playerMux sync.RWMutex                // Protects access to player maps

	// Game state
	ballX, ballY   float64 // Position of the center of the ball
	ballVX, ballVY float64 // Ball velocity

	ticker    *time.Ticker
	stopChan  chan bool // Channel to signal the game loop to stop
	isRunning bool      // Indicates if the game loop is active
}

// NewPongGame creates a new instance of the Pong game.
func NewPongGame(finisher game.GameFinisher, id string) *PongGame {
	return &PongGame{
		gameFinisher: finisher,
		gameID:       id,
		players:      make(map[string]*PongPlayerState),
		playerMap:    make(map[string]game.Player),
		stopChan:     make(chan bool),
		isRunning:    false,
		// Ball position and velocity are set during Reset() in Start()
	}
}

// --- game.Game interface implementation ---

// GetID returns the unique identifier of this game instance.
func (g *PongGame) GetID() string {
	return g.gameID
}

// AddPlayer adds a player to the game, assigning them a role (Player 1 or Player 2).
func (g *PongGame) AddPlayer(player game.Player) error {
	g.playerMux.Lock()
	defer g.playerMux.Unlock()

	if len(g.players) >= MaxPlayers {
		return fmt.Errorf("game %s is full (%d/%d players)", g.gameID, len(g.players), MaxPlayers)
	}

	playerID := player.GetID()
	if _, exists := g.players[playerID]; exists {
		return fmt.Errorf("player %s already in game %s", playerID, g.gameID)
	}

	// Assign role based on current player count
	role := 0
	if len(g.players) == 0 {
		role = 1 // First player is Player 1 (left)
	} else {
		role = 2 // Second player is Player 2 (right)
	}

	// Create the internal player state
	newPlayerState := &PongPlayerState{
		PlayerID: playerID,
		PaddleY:  GameHeight / 2, // Start paddle in the middle
		Score:    0,
		Role:     role,
	}
	g.players[playerID] = newPlayerState
	g.playerMap[playerID] = player // Store the interface for sending messages

	log.Printf("[Game %s] Player %s added as Player %d.", g.gameID, playerID, role)
	return nil
}

// RemovePlayer removes a player from the game. If this causes the player count
// to drop below the minimum, the game is stopped.
func (g *PongGame) RemovePlayer(player game.Player) {
	g.playerMux.Lock()

	playerID := player.GetID()
	_, exists := g.players[playerID]
	if !exists {
		g.playerMux.Unlock()
		log.Printf("[Game %s] Attempted to remove player %s who is not in the game.", g.gameID, playerID)
		return
	}

	role := g.players[playerID].Role
	delete(g.players, playerID)
	delete(g.playerMap, playerID)
	playerCount := len(g.players) // Get count after deletion

	g.playerMux.Unlock() // Unlock before potentially stopping

	log.Printf("[Game %s] Player %s (Player %d) removed.", g.gameID, playerID, role)

	// If the game was running and now has too few players, stop it.
	if g.isRunning && playerCount < MinPlayers {
		log.Printf("[Game %s] Not enough players remaining (%d/%d). Stopping game.", g.gameID, playerCount, MinPlayers)
		// Stop the game asynchronously to avoid deadlocks if called from within game loop context.
		go g.Stop()
	}
}

// Start begins the game loop if the correct number of players are present.
func (g *PongGame) Start() {
	g.playerMux.Lock()
	if len(g.players) != MinPlayers {
		g.playerMux.Unlock()
		log.Printf("[Game %s] Cannot start, requires %d players, but has %d.", g.gameID, MinPlayers, len(g.players))
		// Ensure game is stopped and hub is notified even if start fails pre-loop
		g.Stop() // Stop will handle the !isRunning case gracefully
		return
	}

	// Only proceed if not already running to prevent multiple loops
	if g.isRunning {
		g.playerMux.Unlock()
		log.Printf("[Game %s] Attempted to start game, but it is already running.", g.gameID)
		return
	}

	g.isRunning = true
	g.Reset()                                        // Set initial ball and paddle positions/velocities
	g.ticker = time.NewTicker(16 * time.Millisecond) // ~60 FPS
	g.playerMux.Unlock()

	log.Printf("[Game %s] Starting game loop.", g.gameID)

	// Defer cleanup actions for when the loop exits
	defer func() {
		if g.ticker != nil {
			g.ticker.Stop()
		}
		log.Printf("[Game %s] Game loop stopped.", g.gameID)
		// Notification to the hub happens within the Stop() method.
	}()

	// Main game loop
	for {
		select {
		case <-g.ticker.C:
			// If the game should no longer be running, exit the loop.
			if !g.isRunning {
				return
			}

			g.playerMux.Lock() // Lock for update/send/checkOver
			g.update()         // Update game state (ball, collisions)
			g.sendGameState()  // Send current state to players

			gameOver, winnerID, score1, score2 := g.checkGameOver() // Check win condition
			g.playerMux.Unlock()                                    // Unlock after checks

			if gameOver {
				log.Printf("[Game %s] Game over condition met. Winner: %s, Score: %d-%d", g.gameID, winnerID, score1, score2)
				// Send final game over message before stopping
				g.sendGameOver(winnerID, score1, score2)
				// Stop the game and notify the hub
				g.Stop()
				return // Exit the game loop goroutine
			}

		case <-g.stopChan:
			// Received signal to stop, exit the loop.
			return
		}
	}
}

// Stop gracefully shuts down the game loop and notifies the hub.
func (g *PongGame) Stop() {
	g.playerMux.Lock()
	// Prevent multiple stops or stopping a non-running game.
	if !g.isRunning {
		// Ensure stopChan is closed even if Start() failed early
		select {
		case <-g.stopChan: // Already closed
		default:
			close(g.stopChan)
		}
		g.playerMux.Unlock()
		// If Stop is called before Start completes, notify Hub immediately
		if g.gameFinisher != nil {
			log.Printf("[Game %s] Stopping game that was not fully started.", g.gameID)
			result := game.GameResult{Scores: make(map[string]int)} // Empty result
			// Ensure gameFinisher is called outside the lock
			finisher := g.gameFinisher
			go finisher.GameFinished(g.gameID, result) // Notify asynchronously
		}
		return
	}

	g.isRunning = false
	// Close stopChan safely within the lock context when stopping a running game.
	select {
	case <-g.stopChan: // Already closed (shouldn't happen here if isRunning was true)
	default:
		close(g.stopChan)
	}

	// Get final player states before calculating results
	finalScores := make(map[string]int)
	for pid, pstate := range g.players {
		finalScores[pid] = pstate.Score
	}
	finisher := g.gameFinisher // Copy finisher to call outside lock

	g.playerMux.Unlock() // Unlock before calling finisher

	log.Printf("[Game %s] Stopping game.", g.gameID)

	// Prepare results for the hub
	result := game.GameResult{
		Scores: finalScores, // Provide final scores per PlayerID
	}

	// Notify the hub that the game has finished
	if finisher != nil {
		// Call GameFinished asynchronously to avoid blocking Stop() if hub processing is slow
		// and prevent potential deadlocks if GameFinished tries to lock game resources.
		go finisher.GameFinished(g.gameID, result)
	} else {
		log.Printf("[Game %s] Error: gameFinisher is nil during Stop(). Hub will not be notified.", g.gameID)
	}
}

// HandleMessage processes incoming messages from players during the game.
func (g *PongGame) HandleMessage(player game.Player, msg message.Message) {
	// Only process messages if the game is running.
	if !g.isRunning {
		return
	}

	playerID := player.GetID()

	switch msg.Type {
	case message.PongInput:
		var payload PongInputPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("[Game %s] Error unmarshalling PongInput from %s: %v", g.gameID, playerID, err)
			return
		}

		g.playerMux.Lock()
		pState, ok := g.players[playerID]
		if ok {
			// Update paddle position based on input direction
			newY := pState.PaddleY
			if payload.Direction == "up" {
				newY += PaddleSpeed
			} else if payload.Direction == "down" {
				newY -= PaddleSpeed
			}
			// Clamp paddle position within game boundaries (using center Y)
			halfPaddle := PaddleHeight / 2
			pState.PaddleY = math.Max(halfPaddle, math.Min(GameHeight-halfPaddle, newY))
			// log.Printf("[Game %s] Player %s paddle moved to %.2f", g.gameID, playerID, pState.PaddleY)
		} else {
			log.Printf("[Game %s] Received input from player %s who is not in the internal state map.", g.gameID, playerID)
		}
		g.playerMux.Unlock()

	default:
		log.Printf("[Game %s] Received unhandled message type '%s' from player %s", g.gameID, msg.Type, playerID)
	}
}

// --- Core Game Logic Methods ---

// update advances the game state by one tick, handling ball movement and collisions.
// This method requires the playerMux to be locked by the caller.
func (g *PongGame) update() {
	// 1. Move the ball
	g.ballX += g.ballVX
	g.ballY += g.ballVY

	halfBall := BallSize / 2

	// 2. Check for collisions with top/bottom walls
	if g.ballY-halfBall <= 0 { // Hit bottom wall
		g.ballY = halfBall // Clamp position
		g.ballVY = -g.ballVY
	} else if g.ballY+halfBall >= GameHeight { // Hit top wall
		g.ballY = GameHeight - halfBall // Clamp position
		g.ballVY = -g.ballVY
	}

	// 3. Check for collisions with paddles
	var player1State, player2State *PongPlayerState
	for _, pState := range g.players {
		if pState.Role == 1 {
			player1State = pState
		} else if pState.Role == 2 {
			player2State = pState
		}
	}

	// Ensure both players exist before checking paddles
	if player1State == nil || player2State == nil {
		log.Printf("[Game %s] Error: Player state missing during update.", g.gameID)
		return // Cannot proceed without both players
	}

	halfPaddleH := PaddleHeight / 2

	// Collision with Player 1's paddle (left)
	paddle1LeftEdge := PaddleWidth
	if g.ballVX < 0 && g.ballX-halfBall <= paddle1LeftEdge { // Ball is moving left and near/past the paddle's front edge
		paddle1Top := player1State.PaddleY + halfPaddleH
		paddle1Bottom := player1State.PaddleY - halfPaddleH
		if g.ballY <= paddle1Top && g.ballY >= paddle1Bottom { // Vertical alignment check
			g.ballX = paddle1LeftEdge + halfBall // Clamp ball position to prevent sticking
			g.ballVX = -g.ballVX                 // Reverse horizontal direction
			// Optional: Adjust vertical velocity based on where the ball hit the paddle
			// deltaY := g.ballY - player1State.PaddleY
			// g.ballVY += deltaY * 0.1 // Example adjustment factor
			// Optional: Increase ball speed slightly
			g.increaseBallSpeed()
			// log.Printf("[Game %s] Ball hit Player 1 paddle. New VX: %.2f", g.gameID, g.ballVX)
		}
	}

	// Collision with Player 2's paddle (right)
	paddle2RightEdge := GameWidth - PaddleWidth
	if g.ballVX > 0 && g.ballX+halfBall >= paddle2RightEdge { // Ball is moving right and near/past the paddle's front edge
		paddle2Top := player2State.PaddleY + halfPaddleH
		paddle2Bottom := player2State.PaddleY - halfPaddleH
		if g.ballY <= paddle2Top && g.ballY >= paddle2Bottom { // Vertical alignment check
			g.ballX = paddle2RightEdge - halfBall // Clamp ball position
			g.ballVX = -g.ballVX                  // Reverse horizontal direction
			// Optional: Adjust vertical velocity
			// deltaY := g.ballY - player2State.PaddleY
			// g.ballVY += deltaY * 0.1
			// Optional: Increase ball speed slightly
			g.increaseBallSpeed()
			// log.Printf("[Game %s] Ball hit Player 2 paddle. New VX: %.2f", g.gameID, g.ballVX)
		}
	}

	// 4. Check for scoring (ball hitting left/right walls)
	if g.ballX-halfBall <= 0 { // Ball hit left wall
		player2State.Score++ // Player 2 scores
		log.Printf("[Game %s] Player 2 scored! Score: %d-%d", g.gameID, player1State.Score, player2State.Score)
		g.Reset() // Reset ball and paddles for the next round
	} else if g.ballX+halfBall >= GameWidth { // Ball hit right wall
		player1State.Score++ // Player 1 scores
		log.Printf("[Game %s] Player 1 scored! Score: %d-%d", g.gameID, player1State.Score, player2State.Score)
		g.Reset() // Reset ball and paddles for the next round
	}
}

// increaseBallSpeed slightly increases the ball's speed, capping at max values.
// This method requires the playerMux to be locked by the caller.
func (g *PongGame) increaseBallSpeed() {
	newVX := g.ballVX * SpeedIncrease
	newVY := g.ballVY * SpeedIncrease

	// Apply caps, preserving sign
	if math.Abs(newVX) > MaxBallSpeedX {
		newVX = math.Copysign(MaxBallSpeedX, newVX)
	}
	if math.Abs(newVY) > MaxBallSpeedY {
		newVY = math.Copysign(MaxBallSpeedY, newVY)
	}

	g.ballVX = newVX
	g.ballVY = newVY
}

// checkGameOver determines if the game has ended based on scores.
// Returns gameOver status, winner ID, and final scores.
// This method requires the playerMux to be locked by the caller.
func (g *PongGame) checkGameOver() (gameOver bool, winnerID string, score1 int, score2 int) {
	var p1State, p2State *PongPlayerState
	for _, pState := range g.players {
		if pState.Role == 1 {
			p1State = pState
		} else if pState.Role == 2 {
			p2State = pState
		}
	}

	// Should not happen if game started correctly, but check for safety
	if p1State == nil || p2State == nil {
		return false, "", 0, 0
	}

	score1 = p1State.Score
	score2 = p2State.Score

	if score1 >= TargetScore {
		return true, p1State.PlayerID, score1, score2
	}
	if score2 >= TargetScore {
		return true, p2State.PlayerID, score1, score2
	}

	return false, "", score1, score2
}

// sendGameState broadcasts the current game state to all connected players.
// This method requires the playerMux to be locked by the caller.
func (g *PongGame) sendGameState() {
	var p1State, p2State *PongPlayerState
	for _, pState := range g.players {
		if pState.Role == 1 {
			p1State = pState
		} else if pState.Role == 2 {
			p2State = pState
		}
	}

	// If player states are missing (e.g., during setup/teardown), don't send.
	if p1State == nil || p2State == nil {
		return
	}

	// Create the state payload using data from the assigned roles.
	statePayload := PongStatePayload{
		BallX:    g.ballX,
		BallY:    g.ballY,
		Paddle1Y: p1State.PaddleY,
		Paddle2Y: p2State.PaddleY,
		Score1:   p1State.Score,
		Score2:   p2State.Score,
	}

	// Send the state to all players currently in the game map.
	for playerID, player := range g.playerMap {
		err := player.SendMessage(message.PongState, statePayload)
		if err != nil {
			// Log error, hub's unregister mechanism should handle disconnects.
			log.Printf("[Game %s] Error sending state to player %s: %v", g.gameID, playerID, err)
		}
	}
}

// sendGameOver sends the final game over message to all players.
// This is typically called just before Stop() notifies the hub.
func (g *PongGame) sendGameOver(winnerID string, score1, score2 int) {
	gameOverPayload := PongGameOverPayload{
		Winner: winnerID, // PlayerID of the winner
		Score1: score1,
		Score2: score2,
	}

	g.playerMux.RLock() // Use RLock as we are only reading playerMap
	playersToSend := make([]game.Player, 0, len(g.playerMap))
	for _, p := range g.playerMap {
		playersToSend = append(playersToSend, p)
	}
	g.playerMux.RUnlock() // Release lock before sending

	log.Printf("[Game %s] Sending game over message. Winner: %s, Score: %d-%d", g.gameID, winnerID, score1, score2)
	for _, player := range playersToSend {
		err := player.SendMessage(message.PongGameOver, gameOverPayload)
		if err != nil {
			log.Printf("[Game %s] Error sending game over to player %s: %v", g.gameID, player.GetID(), err)
		}
	}
}

// Reset sets the ball and paddles to their starting positions and assigns
// a random initial velocity to the ball.
// This method requires the playerMux to be locked by the caller.
func (g *PongGame) Reset() {
	// Center the ball
	g.ballX = GameWidth / 2
	g.ballY = GameHeight / 2

	// Assign random initial horizontal direction
	vx := InitialBallVX
	if rand.Intn(2) == 0 {
		vx = -vx
	}
	// Assign random initial vertical direction
	vy := InitialBallVY
	if rand.Intn(2) == 0 {
		vy = -vy
	}
	g.ballVX = vx
	g.ballVY = vy

	// Reset paddle positions
	for _, pState := range g.players {
		pState.PaddleY = GameHeight / 2
	}
	log.Printf("[Game %s] Round reset. Ball velocity: (%.2f, %.2f)", g.gameID, g.ballVX, g.ballVY)
}

// --- Ensure PongGame implements game.Game ---
var _ game.Game = (*PongGame)(nil)
