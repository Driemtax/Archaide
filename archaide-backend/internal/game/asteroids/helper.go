package asteroids

import (
	"math"

	"github.com/Driemtax/Archaide/internal/component"
)

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

func wrapPosition(pos component.Vector2D) component.Vector2D {
	if pos.X < 0 {
		pos.X += WORLD_WIDTH
	} else if pos.X >= WORLD_WIDTH {
		pos.X -= WORLD_WIDTH
	}
	if pos.Y < 0 {
		pos.Y += WORLD_HEIGHT
	} else if pos.Y >= WORLD_HEIGHT {
		pos.Y -= WORLD_HEIGHT
	}
	return pos
}

func checkCollision(pos1, pos2 component.Vector2D, r1, r2 float64) bool {
	distSq := pos1.Sub(pos2).LengthSq()
	radiiSumSq := (r1 + r2) * (r1 + r2)
	return distSq <= radiiSumSq
}

// Helper to find String in Slice
func findString(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}
