import { useWebSocketContext } from "../../hooks/useWebSocketContext";
import { ClientMessage, AsteroidsPlayerMove } from "../../types";

export default function Asteroids() {
  const { sendMessage, asteroidState } = useWebSocketContext();

  const sendDirection = (dir: AsteroidsPlayerMove) => {
    const message: ClientMessage = {
      type: "asteroids_input",
      payload: {
        direction: dir,
      },
    };

    sendMessage(message);
  };

  return (
    <div>
      <h1>Some Game</h1>
      <button onClick={() => sendDirection("north")}>North</button>
      <button onClick={() => sendDirection("east")}>East</button>
      <button onClick={() => sendDirection("south")}>South</button>
      <button onClick={() => sendDirection("west")}>West</button>
      <h2>Asteroids Game State</h2>
      <p>{JSON.stringify(asteroidState)}</p>
    </div>
  );
}
