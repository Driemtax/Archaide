import { extend } from "@pixi/react";
import { Container, Graphics, Texture, Sprite } from "pixi.js";
import * as PIXI from "pixi.js";
import { COLORS, SCREEN } from "./config";
import { AsteroidsAsteroidState } from "../../types";

extend({ Container, Graphics, Sprite });

function getAsteroidWidth(a: AsteroidsAsteroidState) {
  switch (a.type) {
    case "large":
      return 30;
    case "middle":
      return 18;
    case "small":
      return 10;
  }
}

interface AsteroidProps {
  state: AsteroidsAsteroidState;
  rotation: number;
}

export default function Asteroid(props: AsteroidProps) {
  const { state, rotation } = props;

  const variantIndex =
    state.variantIndex !== undefined ? state.variantIndex : 0;
  const assetPath = `assets/sprite_${state.type}_asteroid${variantIndex}.png`;
  const texture = PIXI.Assets.get<Texture>(assetPath);

  if (!texture) {
    console.warn(`Texture not found for ${assetPath}. Falling back to circle.`);
    return (
      <pixiGraphics
        key={`asteroid-fallback-${state.id}`}
        draw={(g) => {
          g.clear();
          g.fill(COLORS.white);
          g.circle(0, 0, getAsteroidWidth(state) * SCREEN.scaling_factor);
          g.fill();
        }}
        x={state.pos.x * SCREEN.scaling_factor}
        y={state.pos.y * SCREEN.scaling_factor}
      />
    );
  }

  return (
    <pixiSprite
      key={`asteroid-${state.id}`}
      texture={texture}
      x={state.pos.x * SCREEN.scaling_factor}
      y={state.pos.y * SCREEN.scaling_factor}
      anchor={0.5}
      scale={SCREEN.scaling_factor}
      rotation={rotation}
    />
  );
}
