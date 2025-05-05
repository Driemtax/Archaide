// Define payloads for specific message types
export interface WelcomePayload {
  clientId: string;
  currentGames?: string[];
}

export interface PlayerInfo {
  score: number;
  inGame: boolean;
}

export interface UpdateLobbyPayload {
  players: Record<string, PlayerInfo>;
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

export interface Vector2D {
  x: number;
  y: number;
}

export interface AsteroidsPlayerState {
  pos: Vector2D;
  dir: Vector2D;
}

export interface AsteroidsStatePayload {
  players: Record<string, AsteroidsPlayerState>;
}

export type AsteroidsPlayerMove =
  | "north"
  | "east"
  | "south"
  | "west"
  | "north_east"
  | "north_west"
  | "south_west"
  | "south_east"
  | "none";

export interface AsteroidsInputPayload {
  direction: AsteroidsPlayerMove;
}

export type ServerMessage =
  | { type: "welcome"; payload: WelcomePayload }
  | { type: "update_lobby"; payload: UpdateLobbyPayload }
  | { type: "game_selected"; payload: GameSelectedPayload }
  | { type: "error"; payload: ErrorPayload }
  | { type: string; payload: unknown } // Fallback for unhandled/generic types
  | { type: "pong_state"; payload: PongStatePayload };

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

export interface AsteroidsInputMessage extends ClientMessageBase {
  type: "asteroids_input";
  payload: AsteroidsInputPayload;
}

export type ClientMessage = ClientSelectGameMessage | AsteroidsInputMessage;
