import { extend } from "@pixi/react";
import { Container } from "pixi.js";
import { useWebSocketContext } from "../../hooks/useWebSocketContext";
import Asteroid from "./Asteroid";
import Player from "./Player";
import Projectile from "./Projectile";

extend({ Container });

export default function AsteroidsStage() {
  const { asteroidState } = useWebSocketContext();

  return (
    <pixiContainer>
      {Object.entries(asteroidState?.players || {}).map(([, p]) => (
        <Player state={p} />
      ))}
      {asteroidState?.projectiles?.map((p) => <Projectile state={p} />)}
      {asteroidState?.asteroids?.map((a) => <Asteroid state={a} />)}
    </pixiContainer>
  );
}
