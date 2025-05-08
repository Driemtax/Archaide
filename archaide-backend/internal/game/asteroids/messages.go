package asteroids

import "github.com/Driemtax/Archaide/internal/component"

// Always tells the server if the button is currently pressed or not
type AsteroidsInputPayload struct {
	Left  bool `json:"left"`
	Right bool `json:"right"`
	Up    bool `json:"up"`
	Shoot bool `json:"shoot"`
}

type PlayerState struct {
	ID           string             `json:"id"`
	Pos          component.Vector2D `json:"pos"`
	Dir          component.Vector2D `json:"dir"`
	Health       float64            `json:"health"`
	IsInvincible bool               `json:"isInvincible"`
	Score        int                `json:"score"`
}

type AsteroidState struct {
	ID           string             `json:"id"`
	Pos          component.Vector2D `json:"pos"`
	Dir          component.Vector2D `json:"dir"`
	VariantIndex int                `json:"variantIndex"`
	Typ          AsteroidType       `json:"type"`
}

type ProjectileState struct {
	ID  string             `json:"id"`
	Pos component.Vector2D `json:"pos"`
}

type AsteroidsStatePayload struct {
	Players     map[string]PlayerState `json:"players"`
	Asteroids   []AsteroidState        `json:"asteroids"`
	Projectiles []ProjectileState      `json:"projectiles"`
}

type AsteroidsGameOverPayload struct {
	Winner string `json:"winner"`
}
