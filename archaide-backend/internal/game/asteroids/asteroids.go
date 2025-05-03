package asteroids

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Driemtax/Archaide/internal/component"
	"github.com/Driemtax/Archaide/internal/game"
	"github.com/Driemtax/Archaide/internal/message"
)

const (
	INITIAL_PLAYER_SPEED float64 = 5.0
)

type AsteroidsPlayer struct {
	Pos      component.Vector2D `json:"pos"`
	Speed    float64            `json:"speed"`
	Dir      component.Vector2D `json:"dir"`
	PlayerID string             // Saving the id of the game.Player aka Client
}

type AsteroidsGame struct {
	// Feels hacky but seems to be a valid practice to remove import cycles from the code
	// But anyways we are getting an interface to notify the hub
	gameFinisher game.GameFinisher

	gameID     string
	players    map[string]*AsteroidsPlayer // Map Player Id to AsteroidPlayer State
	playerMap  map[string]game.Player      // Map Player Id to game.Player aka Client
	playerMux  sync.RWMutex
	ticker     *time.Ticker
	stopChan   chan bool
	isRunning  bool
	minPlayers int
	maxPlayers int
}

func NewAsteroidsGame(finisher game.GameFinisher, id string) *AsteroidsGame {
	return &AsteroidsGame{
		gameFinisher: finisher,
		gameID:       id,
		players:      make(map[string]*AsteroidsPlayer),
		playerMap:    make(map[string]game.Player),
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

	newPlayer := &AsteroidsPlayer{
		Pos:      component.NewVector2D(0, 0),
		Speed:    INITIAL_PLAYER_SPEED,
		Dir:      component.NewVector2D(0, -1),
		PlayerID: playerID,
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
	g.ticker = time.NewTicker(32 * time.Millisecond) // ~30 FPS
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

			g.playerMux.Lock()
			g.update()
			g.sendGameState()
			gameOver := g.checkGameOver() // internal check
			g.playerMux.Unlock()

			if gameOver {
				log.Printf("[Game %s] Game over condition met.", g.gameID)
				// First send the game over message
				g.sendGameOver()
				// Then stop the game.
				// Thats important, otherwise the game gets stopped before the message will be sent
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

	// closing the stop channel
	select {
	case <-g.stopChan: // Already closed
	default:
		close(g.stopChan)
	}

	playersSnapshot := make([]game.Player, 0, len(g.playerMap))
	for _, p := range g.playerMap {
		playersSnapshot = append(playersSnapshot, p)
	}
	g.playerMux.Unlock()

	log.Printf("[Game %s] Stopping game.", g.gameID)

	// TODO: Implement the logic for the scores!
	// Idea: Every player gets points for the time he survived
	result := game.GameResult{
		Scores: make(map[string]int), // Key: PlayerID
	}

	// Inform the hub that the game is finished and retrieve all
	// players back to the lobby
	g.gameFinisher.GameFinished(g.gameID, result)
}

func (g *AsteroidsGame) HandleMessage(player game.Player, msg message.Message) {
	// playerID := player.GetID()

	// switch msg.Type {
	// case message.AsteroidsInput:
	// 	var payload AsteroidsInputPayload
	// 	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
	// 		log.Printf("[Game %s] Error unmarshalling AsteroidsInput from %s: %v", g.gameID, playerID, err)
	// 		return
	// 	}

	// 	g.playerMux.Lock()
	// 	pState, ok := g.players[playerID]
	// 	if ok {
	// 		pState.UpdateDir(payload.Direction)
	// 	} else {
	// 		log.Printf("[Game %s] Received input from player %s who is not in the internal state map.", g.gameID, playerID)
	// 	}
	// 	g.playerMux.Unlock()
	// default:
	// 	log.Printf("[Game %s] Received unhandled message type '%s' from player %s", g.gameID, msg.Type, playerID)
	// }
}

/// --- Finished implementing the game.Game interface ---

func (g *AsteroidsGame) checkGameOver() bool {
	return false
}

func (g *AsteroidsGame) sendGameState() {
	playerStates := make(map[string]AsteroidsPlayerState) // Key: PlayerID
	for pID, pState := range g.players {
		playerStates[pID] = AsteroidsPlayerState{
			Pos: pState.Pos,
			Dir: pState.Dir,
		}
	}

	gameStatePayload := AsteroidsStatePayload{
		Players: playerStates,
	}

	// Send to each player using the saved player interface
	for pID, p := range g.playerMap {
		err := p.SendMessage(message.AsteroidsState, gameStatePayload)
		if err != nil {
			log.Printf("[Game %s] Error sending state to player %s: %v", g.gameID, pID, err)
		}
	}
}

func (g *AsteroidsGame) sendGameOver() {
	g.playerMux.RLock()
	defer g.playerMux.RUnlock()

	winnerID := "no_one_so_far"

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

func (g *AsteroidsGame) update() {
	// stub
}
