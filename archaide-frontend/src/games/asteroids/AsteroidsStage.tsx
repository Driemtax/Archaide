import { extend, useTick } from "@pixi/react";
import { Container } from "pixi.js";
import { useWebSocketContext } from "../../hooks/useWebSocketContext";
import Asteroid from "./Asteroid";
import Player from "./Player";
import Projectile from "./Projectile";
import { useRef } from "react";

extend({ Container });

export default function AsteroidsStage() {
  const { asteroidState, myClientId } = useWebSocketContext();
  const rotation = useRef({
    value: 0,
  });

  useTick((ticker) => {
    rotation.current.value += 0.01 * ticker.deltaTime;
  });

  return (
    <pixiContainer>
      {Object.entries(asteroidState?.players || {}).map(([, p]) => (
        <Player state={p} clientID={myClientId} />
      ))}
      {asteroidState?.projectiles?.map((p) => <Projectile state={p} />)}
      {asteroidState?.asteroids?.map((a) => (
        <Asteroid state={a} rotation={rotation.current.value} />
      ))}
    </pixiContainer>
  );
}
