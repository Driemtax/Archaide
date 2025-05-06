import AsteroidsGame from "../games/asteroids/AsteroidsGame";
import PongGame from "../games/pong/Pong";
import { useWebSocketContext } from "../hooks/useWebSocketContext";

export default function GameView() {
  const { selectedGame } = useWebSocketContext();

  return selectedGame === "Asteroids" ? <AsteroidsGame /> : <PongGame />;
}
