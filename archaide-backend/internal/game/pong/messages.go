package pong

type PongInputPayload struct {
	Direction string `json:"direction"`
}

// PongStatePayload defines the data sent to clients each tick.
type PongStatePayload struct {
	BallX    float64 `json:"ball_x"`
	BallY    float64 `json:"ball_y"`
	Paddle1Y float64 `json:"paddle_1_y"` // Position of player assigned role 1
	Paddle2Y float64 `json:"paddle_2_y"` // Position of player assigned role 2
	Score1   int     `json:"score_1"`    // Score of player assigned role 1
	Score2   int     `json:"score_2"`    // Score of player assigned role 2
}

// PongGameOverPayload defines the message sent when the game ends.
type PongGameOverPayload struct {
	Winner string `json:"winner"`  // PlayerID of the winner, or specific indicator for draw/error
	Score1 int    `json:"score_1"` // Final score for player 1
	Score2 int    `json:"score_2"` // Final score for player 2
}
