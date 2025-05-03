package asteroids

import "github.com/Driemtax/Archaide/internal/component"

type PlayerInputMovement string

const (
	North     PlayerInputMovement = "north"
	East      PlayerInputMovement = "east"
	South     PlayerInputMovement = "south"
	West      PlayerInputMovement = "west"
	NorthEast PlayerInputMovement = "north_east"
	NorthWest PlayerInputMovement = "north_west"
	SouthWest PlayerInputMovement = "south_west"
	SouthEast PlayerInputMovement = "south_east"
)

type AsteroidsInputPayload struct {
	Direction PlayerInputMovement `json:"direction"`
}

type AsteroidsPlayerState struct {
	Pos component.Vector2D `json:"pos"`
	Dir component.Vector2D `json:"dir"`
}

type AsteroidsStatePayload struct {
	Players map[string]AsteroidsPlayerState `json:"players"`
}

type AsteroidsGameOverPayload struct {
	Winner string `json:"winner"`
}
