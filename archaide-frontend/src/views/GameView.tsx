import Asteroids from "../games/asteroids/asteroids";
import { useWebSocketContext } from "../hooks/useWebSocketContext";

export default function GameView() {
  const { selectedGame } = useWebSocketContext();

  return selectedGame === "Asteroids" ? <Asteroids /> : <h1>Pong</h1>;
}
