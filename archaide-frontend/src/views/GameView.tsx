import Asteroids from "../games/asteroids/asteroids";
import PongGame from "../games/pong/Pong";
import { useWebSocketContext } from "../hooks/useWebSocketContext";

export default function GameView() {
  const { selectedGame } = useWebSocketContext();

  return selectedGame === "Asteroids" ? <Asteroids /> : <PongGame />;
}
