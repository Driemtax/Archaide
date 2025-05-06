import { extend } from "@pixi/react";
import { Container, Graphics, Sprite, Texture } from "pixi.js";
import * as PIXI from "pixi.js";
import { COLORS } from "./config";
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
          g.circle(0, 0, 15);
          g.fill();
        }}
        x={state.pos.x - 7.5}
        y={state.pos.y - 7.5}
      />
    );
  }
  return (
    <pixiSprite
      key={`player-${state.id}`}
      texture={texture}
      x={state.pos.x}
      y={state.pos.y}
      rotation={rotation}
      anchor={0.5}
    />
  );
}
