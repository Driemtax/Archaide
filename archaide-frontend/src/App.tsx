import { WebSocketProvider } from "./context/WebSocketContext";
import LobbyView from "./views/LobbyView";
import GameView from "./views/GameView";
import { useWebSocketContext } from "./hooks/useWebSocketContext";

// TODO lets move this into a .env file
const WS_URL = "ws://localhost:3030/ws";

function AppContent() {
  const { selectedGame, readyState, gameError } = useWebSocketContext();

  if (readyState === WebSocket.CONNECTING) {
    return <h1>Connecting to Server...</h1>;
  }

  if (readyState === WebSocket.CLOSED || readyState === WebSocket.CLOSING) {
    return (
      <div>
        Connection lost. Please refresh or wait for reconnect.{" "}
        {gameError && `(${gameError})`}
      </div>
    );
  }

  return <div>{selectedGame ? <GameView /> : <LobbyView />}</div>;
}

export default function App() {
  return (
    <WebSocketProvider url={WS_URL}>
      <AppContent />
    </WebSocketProvider>
  );
}
