import { WebSocketProvider } from "./context/WebSocketContext";
import LobbyView from "./views/LobbyView";
import GameView from "./views/GameView";
import { useWebSocketContext } from "./hooks/useWebSocketContext";
import { Toaster } from "./components/ui/sonner";

const IP = process.env.REACT_APP_IP ? process.env.REACT_APP_IP : "localhost";
const WS_URL = "ws://" + IP + ":3030/ws";

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
      <Toaster />
    </WebSocketProvider>
  );
}
