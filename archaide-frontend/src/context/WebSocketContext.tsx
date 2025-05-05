import {
  createContext,
  useState,
  useCallback,
  useMemo,
  ReactNode,
  useEffect,
} from "react";
import type {
  ServerMessage,
  WelcomePayload,
  UpdateLobbyPayload,
  GameSelectedPayload,
  ErrorPayload,
  ClientMessage,
  PlayerInfo,
  AsteroidsStatePayload,
  PongStatePayload,
} from "../types";
import { useWebSocket } from "../hooks/useWebSocket";

/**
 * Defines the shape of the data and functions provided by the WebSocket context.
 */
interface WebSocketContextState {
  /** The current WebSocket readyState (CONNECTING, OPEN, CLOSING, CLOSED). */
  readyState: number;
  /** The last connection error encountered, or null if none/cleared. */
  connectionError: Event | CloseEvent | null;
  /** Client ID assigned by the server. */
  myClientId: string;
  /** Map of connected player IDs to their info. */
  players: Record<string, PlayerInfo>;
  /** List of games currently available to join or spectate. */
  availableGames: string[];
  /** The game the client is currently participating in or spectating. */
  selectedGame: string | null;
  /** Specific error message received from the server's application logic. */
  gameError: string | null;
  /** State specific to the Asteroids game, null if not playing Asteroids. */
  asteroidState: AsteroidsStatePayload | null;
  /** State specific to the Pong game,. null if not playing Pong */
  pongState: PongStatePayload | null;
  /** Function to send a message (properly typed ClientMessage) to the server. */
  sendMessage: (message: ClientMessage) => void;
  // Note: connect/disconnect functions are removed as connection is now managed by the URL prop.
}

// Create the context with an undefined initial value.
const WebSocketContext = createContext<WebSocketContextState | undefined>(
  undefined,
);

/**
 * Props for the WebSocketProvider component.
 */
interface WebSocketProviderProps {
  url: string | null;
  children: ReactNode;
}

/**
 * Provides WebSocket connectivity and manages shared application state derived from messages.
 */
function WebSocketProvider({ url, children }: WebSocketProviderProps) {
  // --- State managed by the context based on WebSocket messages ---
  const [myClientId, setMyClientId] = useState<string>("");
  const [players, setPlayers] = useState<Record<string, PlayerInfo>>({});
  const [availableGames, setAvailableGames] = useState<string[]>([]);
  const [selectedGame, setSelectedGame] = useState<string | null>(null);
  const [gameError, setGameError] = useState<string | null>(null); // Server logic errors
  const [asteroidState, setAsteroidState] =
    useState<AsteroidsStatePayload | null>(null);
  const [pongState, setPongState] = useState<PongStatePayload | null>(null);

  const resetGameStates = () => {
    setAsteroidState(null);
    setPongState(null);
  };

  const reset = () => {
    resetGameStates();
    setAvailableGames([]);
    setGameError(null);
    setSelectedGame(null);
  };

  // --- WebSocket Message Handling ---
  const handleWebSocketMessage = useCallback((event: MessageEvent<string>) => {
    try {
      const message = JSON.parse(event.data) as ServerMessage;
      console.log(`<- Received:`, message);

      // Clear server-logic errors on receiving any new valid message.
      // Connection errors are handled separately via the hook's error state.
      setGameError(null);

      // Process message based on its type
      switch (message.type) {
        case "welcome": {
          const payload = message.payload as WelcomePayload;
          setMyClientId(payload.clientId);
          // Reset everything just for safety
          // Nothing broke so far
          // Nothing should break
          // But im also breaking nothing with making sure everything is alright
          // So...
          reset();
          setAvailableGames(payload.currentGames ?? []);
          break;
        }
        case "update_lobby": {
          const payload = message.payload as UpdateLobbyPayload;
          setPlayers(payload.players ?? {});
          break;
        }
        case "game_selected": {
          const payload = message.payload as GameSelectedPayload;
          setSelectedGame(payload.selectedGame);
          // Reset states of potentially previously selected games
          resetGameStates();
          break;
        }
        case "asteroids_state": {
          setAsteroidState(message.payload as AsteroidsStatePayload);
          break;
        }
        case "pong_state": {
          setPongState(message.payload as PongStatePayload);
          break;
        }
        case "error": {
          const payload = message.payload as ErrorPayload;
          console.error(`Server Logic Error: ${payload.message}`);
          setGameError(payload.message);
          break;
        }
        // Extend this with every other message that we want and need to handle!
        default:
          console.warn(
            `Unknown message type received: ${(message as ServerMessage).type}`,
          );
      }
    } catch (e) {
      console.error(
        "Error parsing WebSocket message:",
        e,
        "Raw data:",
        event.data,
      );
      setGameError("Received invalid data from server."); // Set a generic error
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // --- Use the WebSocket ---
  const {
    sendMessage: wsSend,
    error: connectionError,
    readyState,
  } = useWebSocket(url, {
    onMessage: handleWebSocketMessage,
  });

  // --- Effect to react to connection status changes from the hook ---
  useEffect(() => {
    if (connectionError) {
      console.error(
        "WebSocket Connection Error/Unexpected Close:",
        connectionError,
      );
      setGameError("Connection issue. Please wait or refresh.");
      // Reset application state that depends on a live connection
      setMyClientId("");
      setPlayers({});
      setAvailableGames([]);
      setSelectedGame(null);
      setAsteroidState(null);
    } else if (readyState === WebSocket.OPEN) {
      if (gameError?.startsWith("Connection issue")) {
        setGameError(null);
      }
    }
    // Dependency: React to changes in the error object from the hook or the readyState.
  }, [connectionError, readyState, gameError]); // Include gameError to allow clearing it

  // --- Send Message Function ---
  const sendMessage = useCallback(
    (message: ClientMessage) => {
      if (readyState === WebSocket.OPEN) {
        console.log(`-> Sending:`, message);
        const messageString = JSON.stringify(message);
        wsSend(messageString);
      } else {
        console.warn("WebSocket not open. Message not sent:", message);
        setGameError("Cannot send message: Not connected.");
      }
    },
    [readyState, wsSend], // Dependencies: readyState to check connection, wsSend to send
  );

  // --- Context Value ---
  const contextValue = useMemo(
    () => ({
      readyState,
      connectionError,
      myClientId,
      players,
      availableGames,
      selectedGame,
      gameError,
      asteroidState,
      pongState,
      sendMessage,
    }),
    [
      readyState,
      connectionError,
      myClientId,
      players,
      availableGames,
      selectedGame,
      gameError,
      asteroidState,
      pongState,
      sendMessage,
    ],
  );

  return (
    <WebSocketContext.Provider value={contextValue}>
      {children}
    </WebSocketContext.Provider>
  );
}

export { WebSocketContext, WebSocketProvider };
