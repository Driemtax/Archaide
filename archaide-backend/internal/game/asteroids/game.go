package asteroids

import (
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/Driemtax/Archaide/internal/component"
	"github.com/google/uuid"
)

// My general idea for this game will be, that
// the player can send his input
// im going to save this input into each player struct
// and then when update is called
// im going to update the player according to the
// new player infos!

// This function determines how much the player is allowed to turn
func (p *Player) HandleInput(i AsteroidsInputPayload) {
	p.LastInput = i
}

func (g *AsteroidsGame) update(dt float64) {
	now := time.Now()

	// Update the players
	for _, p := range g.players {
		if p.Health.IsDead() {
			continue
		}

		// Stop invincibility
		if p.IsInvincible && now.After(p.InvincibleTime) {
			p.IsInvincible = false
			log.Printf("[Game %s] Player %s invincibility ended.", g.gameID, p.PlayerID)
		}

		// Apply Input
		turnDirection := 0.0
		if p.LastInput.Left && !p.LastInput.Right {
			turnDirection = -1.0 // Turn Left (counter-clockwise)
		} else if p.LastInput.Right && !p.LastInput.Left {
			turnDirection = 1.0 // Turn Right (clockwise)
		}

		if turnDirection != 0 {
			angleDelta := p.TurnSpeed * turnDirection * dt
			cos := math.Cos(angleDelta)
			sin := math.Sin(angleDelta)
			newX := p.Dir.X*cos - p.Dir.Y*sin
			newY := p.Dir.X*sin + p.Dir.Y*cos
			p.Dir = component.NewVector2D(newX, newY).Normalize()
		}

		if p.LastInput.Up {
			// TODO lets add some kind of fancy velocity in here later on
			moveStep := p.Dir.Mul(p.Speed * dt)
			p.Pos = p.Pos.Add(moveStep)
		}

		if p.LastInput.Shoot && now.After(p.LastShotTime.Add(PLAYER_SHOOT_COOLDOWN)) {
			g.spawnProjectile(p)
			p.LastShotTime = now
		}

		// Screen Wrapping
		p.Pos = wrapPosition(p.Pos)
	}

	/// --- Update Projectiles ---
	projectilesToRemove := []string{}
	for id, proj := range g.projectiles {
		// Move the projectile
		proj.Pos = proj.Pos.Add(proj.Dir.Mul(proj.Speed * dt))
		// Projectiles are also getting wrapped...
		proj.Pos = wrapPosition(proj.Pos)

		// Check if the lifetime is expired
		if now.Sub(proj.SpawnTime) > PROJECTILE_LIFETIME {
			projectilesToRemove = append(projectilesToRemove, id)
		}
	}
	// Remove expired projectiles
	for _, id := range projectilesToRemove {
		delete(g.projectiles, id)
	}

	/// --- Update Asteroids ---
	for _, ast := range g.asteroids {
		// Move the Asteroid
		ast.Pos = ast.Pos.Add(ast.Dir.Mul(ast.Speed * dt))
		// Wrap the Asteroid Position
		ast.Pos = wrapPosition(ast.Pos)
	}

	/// --- Collision Detection ---
	clearAsteroids := []string{}
	clearProjectiles := []string{}
	asteroidsToAdd := []*Asteroid{}

	// Player vs Asteroid
	for _, p := range g.players {
		if p.IsInvincible || p.Health.IsDead() {
			continue
		}
		for astID, ast := range g.asteroids {
			if checkCollision(p.Pos, ast.Pos, p.Radius, ast.Radius) {
				p.Health.Damage(1)
				g.respawnPlayer(p)
				if _, exists := g.asteroids[astID]; exists {
					clearAsteroids = append(clearAsteroids, astID)
					newAsteroids := g.splitAsteroid(ast)
					asteroidsToAdd = append(asteroidsToAdd, newAsteroids...)
				}
			}
			break
		}
	}

	// Projectile vs Asteroid
	for projID, proj := range g.projectiles {
		if _, marked := findString(clearProjectiles, projID); marked {
			// Skip projectiles that have already been marked for removal
			continue
		}
		for astID, ast := range g.asteroids {
			if _, marked := findString(clearAsteroids, astID); marked {
				// Skip Asteroids that have already been marked for removal
				continue
			}

			if checkCollision(proj.Pos, ast.Pos, proj.Radius, ast.Radius) {
				log.Printf("[Game %s] Projectile %s hit asteroid %s!", g.gameID, projID, astID)

				clearProjectiles = append(clearProjectiles, projID)
				clearAsteroids = append(clearAsteroids, astID)

				// Award score to the owner of the projectile
				if owner, ok := g.players[proj.OwnerID]; ok {
					points := 0
					switch ast.Type {
					case LARGE:
						points = ASTEROID_POINTS_LARGE
					case MIDDLE:
						points = ASTEROID_POINTS_MIDDLE
					case SMALL:
						points = ASTEROID_POINTS_SMALL
					}
					owner.Score += points
					log.Printf("[Game %s] Player %s score: %d (+%d)", g.gameID, owner.PlayerID, owner.Score, points)
				}

				// Split the asteroid if not small
				newAsteroids := g.splitAsteroid(ast)
				asteroidsToAdd = append(asteroidsToAdd, newAsteroids...)

				// Each projectile can only hit one asteroid. Break inner loop.
				break
			}
		}
	}

	/// --- Apply Removals and Additions ---

	for _, id := range clearProjectiles {
		delete(g.projectiles, id)
	}
	for _, id := range clearAsteroids {
		if _, exists := g.asteroids[id]; exists {
			delete(g.asteroids, id)
		}
	}
	for _, ast := range asteroidsToAdd {
		g.asteroids[ast.ID] = ast
	}

	/// --- Spawn new Asteroids ---
	// If there are not enough asteroids left, spawn more
	if len(g.asteroids) < INITIAL_ASTEROID_COUNT/2 && len(g.players) > 0 {
		// Spawn one new large asteroid at edge
		edge := rand.IntN(4) // 0: top, 1: bottom, 2: left, 3: right
		var spawnPos component.Vector2D
		switch edge {
		case 0:
			spawnPos = component.NewVector2D(rand.Float64()*WORLD_WIDTH, -ASTEROID_SPAWN_PADDING)
		case 1:
			spawnPos = component.NewVector2D(rand.Float64()*WORLD_WIDTH, WORLD_HEIGHT+ASTEROID_SPAWN_PADDING)
		case 2:
			spawnPos = component.NewVector2D(-ASTEROID_SPAWN_PADDING, rand.Float64()*WORLD_HEIGHT)
		case 3:
			spawnPos = component.NewVector2D(WORLD_WIDTH+ASTEROID_SPAWN_PADDING, rand.Float64()*WORLD_HEIGHT)
		}
		log.Printf("[Game %s] Asteroid count low, spawning new one.", g.gameID)
		g.spawnAsteroid(spawnPos, LARGE)
	}
}

func (g *AsteroidsGame) initializeAsteroids() {
	log.Printf("[Game %s] Initializing %d asteroids.", g.gameID, INITIAL_ASTEROID_COUNT)
	center := component.NewVector2D(WORLD_WIDTH/2, WORLD_HEIGHT/2)
	for range INITIAL_ASTEROID_COUNT {
		// Spawn asteroids away from the center
		angle := rand.Float64() * 2 * math.Pi
		dist := ASTEROID_SPAWN_PADDING + rand.Float64()*(math.Min(WORLD_WIDTH, WORLD_HEIGHT)/2-ASTEROID_SPAWN_PADDING)
		pos := center.Add(component.NewVector2D(math.Cos(angle)*dist, math.Sin(angle)*dist))

		g.spawnAsteroid(pos, LARGE)
	}
}

func (g *AsteroidsGame) spawnAsteroid(pos component.Vector2D, typ AsteroidType) *Asteroid {
	id := uuid.NewString()
	dir := component.NewVector2D(rand.Float64()*2-1, rand.Float64()*2-1).Normalize()
	if dir.LengthSq() == 0 { // Avoid zero vector
		dir = component.NewVector2D(1, 0)
	}
	speed := ASTEROID_SPEED_MIN + rand.Float64()*(ASTEROID_SPEED_MAX-ASTEROID_SPEED_MIN)
	var radius float64

	switch typ {
	case LARGE:
		radius = 30.0
	case MIDDLE:
		radius = 18.0
		speed *= 1.3 // The smaller the asteroid the faster the asteroid
	case SMALL:
		radius = 10.0
		speed *= 1.6
	default:
		log.Printf("[Game %s] Warning: Tried to spawn unknown asteroid type '%s'", g.gameID, typ)
		return nil
	}

	asteroid := &Asteroid{
		ID:           id,
		Pos:          pos,
		Dir:          dir,
		Type:         typ,
		Speed:        speed,
		Radius:       radius,
		VariantIndex: rand.IntN(2),
	}
	g.asteroids[id] = asteroid
	// log.Printf("[Game %s] Spawned asteroid %s (%s) at %.1f, %.1f", g.gameID, id, typ, pos.X, pos.Y)
	return asteroid
}

func (g *AsteroidsGame) spawnProjectile(p *Player) {
	now := time.Now()

	if now.Sub(p.LastShotTime) < PLAYER_SHOOT_COOLDOWN {
		return
	}

	id := uuid.NewString()
	// Carefull im trying to spawn the projectile slightly
	// in front of the player i will maybe have to adjust this
	spawnPos := p.Pos.Add(p.Dir.Mul(p.Radius + PROJECTILE_RADIUS + 1))

	projectile := &Projectile{
		ID:        id,
		OwnerID:   p.PlayerID,
		Pos:       spawnPos,
		Dir:       p.Dir,
		Speed:     PROJECTILE_SPEED,
		SpawnTime: now,
		Radius:    PROJECTILE_RADIUS,
	}
	g.projectiles[id] = projectile
}

func (g *AsteroidsGame) splitAsteroid(original *Asteroid) []*Asteroid {
	newAsteroids := []*Asteroid{}
	var nextType AsteroidType
	canSplit := true

	switch original.Type {
	case LARGE:
		nextType = MIDDLE
	case MIDDLE:
		nextType = SMALL
	case SMALL:
		canSplit = false // Small asteroids can't be splitted further
	default:
		canSplit = false
	}

	if canSplit {
		log.Printf("[Game %s] Splitting asteroid %s (%s) into %d %s asteroids", g.gameID, original.ID, original.Type, ASTEROID_SPLIT_COUNT, nextType)
		baseAngleRad := math.Atan2(original.Dir.Y, original.Dir.X)
		angleVarianceRad := degreesToRadians(ASTEROID_SPLIT_ANGLE_VARY)

		for range ASTEROID_SPLIT_COUNT {
			// Each new angle should get a slightly diffrent angle
			offsetAngle := (rand.Float64()*2 - 1) * angleVarianceRad
			// TODO this here could be an alternative split angle that could be tested
			// offsetAngle := (float64(i)/float64(ASTEROID_SPLIT_COUNT-1) - 0.5) * 2 * angleVarianceRad

			newAngle := baseAngleRad + offsetAngle
			newDir := component.NewVector2D(math.Cos(newAngle), math.Sin(newAngle))

			// Spawn slightly offset from the original position
			spawnOffset := newDir.Mul(original.Radius / 2) // Move slightly outwards
			newPos := original.Pos.Add(spawnOffset)

			spawned := g.spawnAsteroid(newPos, nextType)
			if spawned != nil {
				spawned.Dir = newDir
				newAsteroids = append(newAsteroids, spawned)
			}
		}
	}
	return newAsteroids
}

func (g *AsteroidsGame) respawnPlayer(p *Player) {
	log.Printf("[Game %s] Respawning player %s", g.gameID, p.PlayerID)
	p.Pos = component.NewVector2D(WORLD_WIDTH/2, WORLD_HEIGHT/2) // Respawn at center
	p.Dir = component.NewVector2D(0, -1)
	p.IsInvincible = true
	p.InvincibleTime = time.Now().Add(PLAYER_RESPAWN_INVINCIBLE)
	// TODO i want to implement some fancy velocity later on!
	// for that i will have to reset it here...
}

func (g *AsteroidsGame) determineWinner() string {
	alivePlayers := []*Player{}
	highestScore := -1
	winnerID := ""

	for _, p := range g.players {
		if !p.Health.IsDead() {
			alivePlayers = append(alivePlayers, p)
		}
		if p.Score > highestScore {
			highestScore = p.Score
		}
	}

	if len(alivePlayers) == 1 {
		winnerID = alivePlayers[0].PlayerID
		log.Printf("[Game %s] Game Over. Winner by survival: %s", g.gameID, winnerID)
	} else if len(alivePlayers) == 0 && highestScore >= 0 {
		topScorers := []string{}
		for _, p := range g.players {
			if p.Score == highestScore {
				topScorers = append(topScorers, p.PlayerID)
			}
		}
		if len(topScorers) == 1 {
			winnerID = topScorers[0]
			log.Printf("[Game %s] Game Over. Winner by score (all dead): %s (%d points)", g.gameID, winnerID, highestScore)
		} else {
			log.Printf("[Game %s] Game Over. Draw between players: %v (Score: %d)", g.gameID, topScorers, highestScore)
			winnerID = "draw" // In case of a draw! WARNING this should also be implemented in the frontend!
		}
	} else if len(alivePlayers) > 1 {
		// This case shouldn't happen if checkGameOver triggers Stop() correctly (when <= 1 alive).
		// But if it does (e.g. Stop called manually), determine winner by score among survivors.
		log.Printf("[Game %s] Game Over. Multiple survivors (%d). Determining winner by score.", g.gameID, len(alivePlayers))
		highestScoreAmongSurvivors := -1
		winners := []string{}
		for _, p := range alivePlayers {
			if p.Score > highestScoreAmongSurvivors {
				highestScoreAmongSurvivors = p.Score
				winners = []string{p.PlayerID}
			} else if p.Score == highestScoreAmongSurvivors {
				winners = append(winners, p.PlayerID)
			}
		}
		if len(winners) == 1 {
			winnerID = winners[0]
			log.Printf("[Game %s] Winner by score (survivor): %s (%d points)", g.gameID, winnerID, highestScoreAmongSurvivors)
		} else {
			log.Printf("[Game %s] Draw between survivors: %v (Score: %d)", g.gameID, winners, highestScoreAmongSurvivors)
			winnerID = "draw"
		}
	} else {
		log.Printf("[Game %s] Game Over. No clear winner.", g.gameID)
	}

	return winnerID
}
