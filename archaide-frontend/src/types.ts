// Define payloads for specific message types
export interface WelcomePayload {
  clientId: string;
  currentGames?: string[];
}

export interface UpdateLobbyPayload {
  players: Record<string, number>; // Maps clientId (string) to score (number)
}

export interface GameSelectedPayload {
  selectedGame: string;
}

export interface ErrorPayload {
  message: string;
}

export interface PongStatePayload {
  BallX: number;
  BallY: number;
  Paddle1Y: number;
  Paddle2Y: number;
  Score1: number;
  Score2: number;
}

export type ServerMessage =
  | { type: "welcome"; payload: WelcomePayload }
  | { type: "update_lobby"; payload: UpdateLobbyPayload }
  | { type: "game_selected"; payload: GameSelectedPayload }
  | { type: "error"; payload: ErrorPayload }
  | { type: string; payload: unknown } // Fallback for unhandled/generic types
  | { type: "pong_state"; payload: PongStatePayload};

export interface ClientSelectGamePayload {
  game: string;
}

export interface ClientMessageBase {
  type: string;
  payload?: unknown;
}

export interface ClientSelectGameMessage extends ClientMessageBase {
  type: "select_game";
  payload: ClientSelectGamePayload;
}

export type ClientMessage = ClientSelectGameMessage;


