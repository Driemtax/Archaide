import { extend } from "@pixi/react";
import { Container, Graphics, Sprite, Texture } from "pixi.js";
import * as PIXI from "pixi.js";
import { COLORS, SCREEN } from "./config";
import { AsteroidsPlayerState } from "../../types";

extend({ Container, Graphics, Sprite });

interface PlayerProps {
  state: AsteroidsPlayerState;
}

export default function Player(props: PlayerProps) {
  const { state } = props;

  const assetPath = "assets/sprite_asteroids_player.png";
  const texture = PIXI.Assets.get<Texture>(assetPath);
  const angleFromXAxis = Math.atan2(state.dir.y, state.dir.x);
  const rotation = angleFromXAxis + Math.PI / 2;

  if (!texture) {
    console.warn(`Texture not found for ${assetPath}. Falling back to circle`);
    return (
      <pixiGraphics
        key={`player-${state.id}`}
        draw={(g) => {
          g.clear();
          g.fill(COLORS.white);
          g.circle(0, 0, 15 * SCREEN.scaling_factor);
          g.fill();
        }}
        x={(state.pos.x - 7.5) * SCREEN.scaling_factor}
        y={(state.pos.y - 7.5) * SCREEN.scaling_factor}
      />
    );
  }
  return (
    <pixiSprite
      key={`player-${state.id}`}
      texture={texture}
      x={state.pos.x * SCREEN.scaling_factor}
      y={state.pos.y * SCREEN.scaling_factor}
      rotation={rotation}
      scale={SCREEN.scaling_factor}
      anchor={0.5}
    />
  );
}
