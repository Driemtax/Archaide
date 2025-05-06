package asteroids

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Driemtax/Archaide/internal/component"
	"github.com/Driemtax/Archaide/internal/game"
	"github.com/Driemtax/Archaide/internal/message"
)

const (
	// Player Settings
	INITIAL_PLAYER_SPEED      float64       = 250.0 // Units per secon
	INITIAL_TURN_SPEED_DEG    float64       = 180.0 // Degrees per second
	INITIAL_PLAYER_HEALTH     float64       = 3.0
	PLAYER_RADIUS             float64       = 15.0
	PLAYER_RESPAWN_INVINCIBLE time.Duration = 3 * time.Second
	PLAYER_SHOOT_COOLDOWN     time.Duration = 250 * time.Millisecond

	// Projectile Settings
	PROJECTILE_SPEED    float64       = 400.0 // Units per second
	PROJECTILE_LIFETIME time.Duration = 1500 * time.Millisecond
	PROJECTILE_RADIUS   float64       = 3.0

	// Asteroid Settings
	INITIAL_ASTEROID_COUNT    int     = 8
	ASTEROID_SPAWN_PADDING    float64 = 100.0 // The minimal distance from the center to spawn
	ASTEROID_SPEED_MIN        float64 = 30.0
	ASTEROID_SPEED_MAX        float64 = 80.0
	ASTEROID_POINTS_LARGE     int     = 20
	ASTEROID_POINTS_MIDDLE    int     = 50
	ASTEROID_POINTS_SMALL     int     = 100
	ASTEROID_SPLIT_COUNT      int     = 2  // Into how many pieces an asteroid breaks after getting hit
	ASTEROID_SPLIT_ANGLE_VARY float64 = 30 // The degress of variance for the direction of asteroids after splitting

	// Game World Settings
	WORLD_WIDTH  float64 = 800.0
	WORLD_HEIGHT float64 = 600.0

	// Game Loop
	TICK_RATE time.Duration = 33 * time.Millisecond // ~30 FPS
)

type Player struct {
	Pos            component.Vector2D `json:"pos"`
	Speed          float64            `json:"speed"`
	Dir            component.Vector2D `json:"dir"`
	TurnSpeed      float64
	Health         component.Health
	LastInput      AsteroidsInputPayload
	PlayerID       string // Saving the id of the game.Player aka Client
	Score          int
	LastShotTime   time.Time
	IsInvincible   bool
	InvincibleTime time.Time
	Radius         float64
}

type AsteroidType string

const (
	LARGE  AsteroidType = "large"
	MIDDLE AsteroidType = "middle"
	SMALL  AsteroidType = "small"
)

type Asteroid struct {
	ID     string
	Pos    component.Vector2D
	Dir    component.Vector2D
	Type   AsteroidType
	Speed  float64
	Radius float64
	// Just for the display variant in the frontend
	VariantIndex int
}

type Projectile struct {
	ID        string
	OwnerID   string
	Pos       component.Vector2D
	Dir       component.Vector2D
	Speed     float64
	SpawnTime time.Time
	Radius    float64
}

type AsteroidsGame struct {
	// Feels hacky but seems to be a valid practice to remove import cycles from the code
	// But anyways we are getting an interface to notify the hub
	gameFinisher game.GameFinisher

	gameID       string
	players      map[string]*Player     // Map Player Id to AsteroidPlayer State
	playerMap    map[string]game.Player // Map Player Id to game.Player aka Client
	asteroids    map[string]*Asteroid
	projectiles  map[string]*Projectile
	playerMux    sync.RWMutex
	ticker       *time.Ticker
	stopChan     chan bool
	isRunning    bool
	minPlayers   int
	maxPlayers   int
	lastTickTime time.Time // For my delta time
}

func NewAsteroidsGame(finisher game.GameFinisher, id string) *AsteroidsGame {
	return &AsteroidsGame{
		gameFinisher: finisher,
		gameID:       id,
		players:      make(map[string]*Player),
		playerMap:    make(map[string]game.Player),
		asteroids:    make(map[string]*Asteroid),
		projectiles:  make(map[string]*Projectile),
		stopChan:     make(chan bool),
		isRunning:    false,
		minPlayers:   2,
		maxPlayers:   4,
	}
}

/// --- Implementing the game.Game interface ---

func (g *AsteroidsGame) GetID() string {
	return g.gameID
}

func (g *AsteroidsGame) AddPlayer(player game.Player) error {
	g.playerMux.Lock()
	defer g.playerMux.Unlock()

	if len(g.players) >= g.maxPlayers {
		return fmt.Errorf("game %s is full (%d/%d players)", g.gameID, len(g.players), g.maxPlayers)
	}

	playerID := player.GetID()
	if _, exists := g.players[playerID]; exists {
		return fmt.Errorf("player %s already in game %s", playerID, g.gameID)
	}

	spwanPos := component.NewVector2D(WORLD_WIDTH/2, WORLD_HEIGHT/2)

	newPlayer := &Player{
		Pos:            spwanPos,
		Speed:          INITIAL_PLAYER_SPEED,
		Dir:            component.NewVector2D(0, -1), // Point up
		LastInput:      AsteroidsInputPayload{},
		Health:         component.NewHealth(INITIAL_PLAYER_HEALTH),
		TurnSpeed:      degreesToRadians(INITIAL_PLAYER_SPEED),
		PlayerID:       playerID,
		Score:          0,
		IsInvincible:   true,
		InvincibleTime: time.Now().Add(PLAYER_RESPAWN_INVINCIBLE),
		Radius:         PLAYER_RADIUS,
	}
	g.players[playerID] = newPlayer
	g.playerMap[playerID] = player // Saving the game.Player instance

	log.Printf("[Game %s] Player %s added.", g.gameID, playerID)
	return nil
}

func (g *AsteroidsGame) RemovePlayer(player game.Player) {
	g.playerMux.Lock()
	defer g.playerMux.Unlock()

	playerID := player.GetID()
	if _, ok := g.players[playerID]; ok {
		delete(g.players, playerID)
		delete(g.playerMap, playerID)
		log.Printf("[Game %s] Player %s removed.", g.gameID, playerID)

		if len(g.players) < g.minPlayers && g.isRunning {
			log.Printf("[Game %s] Not enough players remaining (%d/%d). Stopping game.", g.gameID, len(g.players), g.minPlayers)
			// Stopping the game
			// Its important to stop the game inside of a goroutine to not create
			// a deadlock... It looks a bit weird but we need it *sob*
			go g.Stop()
		}
	}
}

func (g *AsteroidsGame) Start() {
	g.playerMux.Lock()
	if len(g.players) < g.minPlayers {
		g.playerMux.Unlock()
		log.Printf("[Game %s] Cannot start, not enough players (%d/%d).", g.gameID, len(g.players), g.minPlayers)
		g.Stop()
		return
	}
	g.isRunning = true
	g.lastTickTime = time.Now()
	g.ticker = time.NewTicker(TICK_RATE)
	g.initializeAsteroids()
	g.playerMux.Unlock()

	log.Printf("[Game %s] Starting game loop.", g.gameID)
	defer func() {
		if g.ticker != nil {
			g.ticker.Stop()
		}
		log.Printf("[Game %s] Game loop stopped.", g.gameID)
		// Calling gameFinisher.GameFinished happens in game.Stop()
	}()

	for {
		select {
		case <-g.ticker.C:
			if !g.isRunning {
				return
			}
			// Calculate Delta Time
			now := time.Now()
			dt := now.Sub(g.lastTickTime).Seconds()
			g.lastTickTime = now

			g.playerMux.Lock()

			g.update(dt)

			gameOver, _ := g.checkGameOver() // internal check

			g.sendGameState()

			g.playerMux.Unlock()
			if gameOver {
				log.Printf("[Game %s] Game over condition met.", g.gameID)
				g.playerMux.RLock()
				winnerID := g.determineWinner()
				g.playerMux.RUnlock()
				g.sendGameOver(winnerID)
				g.Stop()
				return
			}
		case <-g.stopChan:
			// Received a stopping signal from the hub
			// So we stop the go routine1
			return
		}
	}
}

func (g *AsteroidsGame) Stop() {
	g.playerMux.Lock()
	if !g.isRunning {
		g.playerMux.Unlock()
		return
	}
	g.isRunning = false

	if g.ticker != nil {
		g.ticker.Stop()
		g.ticker = nil
	}

	// closing the stop channel
	select {
	case <-g.stopChan: // Already closed
	default:
		close(g.stopChan)
	}

	result := game.GameResult{
		Scores: make(map[string]int),
	}
	for playerID, playerState := range g.players {
		result.Scores[playerID] = playerState.Score
	}

	playersSnapshot := make([]game.Player, 0, len(g.playerMap))
	for _, p := range g.playerMap {
		playersSnapshot = append(playersSnapshot, p)
	}
	g.playerMux.Unlock()

	log.Printf("[Game %s] Stopping game.", g.gameID)

	// Inform the hub that the game is finished and retrieve all
	// players back to the lobby
	g.gameFinisher.GameFinished(g.gameID, result)
}

func (g *AsteroidsGame) HandleMessage(player game.Player, msg message.Message) {
	playerID := player.GetID()

	switch msg.Type {
	case message.AsteroidsInput:
		var payload AsteroidsInputPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("[Game %s] Error unmarshalling AsteroidsInput from %s: %v", g.gameID, playerID, err)
			return
		}

		g.playerMux.Lock()
		pState, ok := g.players[playerID]
		if ok {
			pState.HandleInput(payload)
		} else {
			log.Printf("[Game %s] Received input from player %s who is not in the internal state map.", g.gameID, playerID)
		}
		g.playerMux.Unlock()
	default:
		log.Printf("[Game %s] Received unhandled message type '%s' from player %s", g.gameID, msg.Type, playerID)
	}
}

/// --- Finished implementing the game.Game interface ---

// Checks if the game should end
func (g *AsteroidsGame) checkGameOver() (bool, string) {
	alivePlayers := []string{}
	for playerID, pState := range g.players {
		if !pState.Health.IsDead() {
			alivePlayers = append(alivePlayers, playerID)
		}
	}

	numPlayers := len(g.players)
	numAlive := len(alivePlayers)

	// Game ends if 0 or 1 players are left alive in a multiplayer game,
	// or if the only player dies in a single-player game.
	if numPlayers >= g.minPlayers && numAlive <= 1 {
		if numAlive == 1 {
			return true, alivePlayers[0] // Last one standing wins
		} else {
			return true, "" // All dead, no winner (or handle draw score later)
		}
	}

	// Game doesn't end yet
	return false, ""
}

// Sends the current game state to all connected players
func (g *AsteroidsGame) sendGameState() {
	playerStates := make(map[string]PlayerState)
	for pID, pState := range g.players {
		playerStates[pID] = PlayerState{
			Pos:          pState.Pos,
			Dir:          pState.Dir,
			Health:       pState.Health.HP,
			IsInvincible: pState.IsInvincible,
			Score:        pState.Score,
			ID:           pState.PlayerID,
		}
	}

	asteroidStates := make([]AsteroidState, 0, len(g.asteroids))
	for _, ast := range g.asteroids {
		asteroidStates = append(asteroidStates, AsteroidState{
			ID:           ast.ID,
			Pos:          ast.Pos,
			Dir:          ast.Dir,
			Typ:          ast.Type,
			VariantIndex: ast.VariantIndex,
		})
	}

	projectileStates := make([]ProjectileState, 0, len(g.projectiles))
	for _, proj := range g.projectiles {
		projectileStates = append(projectileStates, ProjectileState{
			ID:  proj.ID,
			Pos: proj.Pos,
		})
	}

	gameStatePayload := AsteroidsStatePayload{
		Players:     playerStates,
		Asteroids:   asteroidStates,
		Projectiles: projectileStates,
	}

	// Send to each player
	payloadBytes, err := json.Marshal(gameStatePayload)
	if err != nil {
		log.Printf("[Game %s] Error marshalling game state: %v", g.gameID, err)
		return
	}

	stateMessage := message.Message{
		Type:    message.AsteroidsState,
		Payload: payloadBytes,
	}

	// fmt.Printf("[Game %s] Sending State: %d players, %d asteroids, %d projectiles\n", g.gameID, len(playerStates), len(asteroidStates), len(projectileStates))

	for pID, p := range g.playerMap {
		if err := p.SendMessage(stateMessage.Type, gameStatePayload); err != nil { // Send the struct directly if SendMessage handles marshalling
			log.Printf("[Game %s] Error sending state to player %s: %v", g.gameID, pID, err)
			// TODO we could consider to build that
			// a player gets removed from a game if sending packages to him
			// fails multiple time
		}
	}
}

func (g *AsteroidsGame) sendGameOver(winnerID string) {
	g.playerMux.RLock()
	defer g.playerMux.RUnlock()

	gameOverPayload := AsteroidsGameOverPayload{
		Winner: winnerID,
	}

	log.Printf("[Game %s] Sending game over message. Winner: %s", g.gameID, winnerID)

	for pID, p := range g.playerMap {
		err := p.SendMessage(message.AsteroidsGameOver, gameOverPayload)
		if err != nil {
			log.Printf("[Game %s] Error sending game over to player %s: %v", g.gameID, pID, err)
		}
	}
}
