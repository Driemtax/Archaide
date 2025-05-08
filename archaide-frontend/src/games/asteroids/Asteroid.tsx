import { extend } from "@pixi/react";
import { Container, Graphics, Texture, Sprite } from "pixi.js";
import * as PIXI from "pixi.js";
import { COLORS } from "./config";
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
}

export default function Asteroid(props: AsteroidProps) {
  const { state } = props;

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
          g.circle(0, 0, getAsteroidWidth(state));
          g.fill();
        }}
        x={state.pos.x}
        y={state.pos.y}
      />
    );
  }

  return (
    <pixiSprite
      key={`asteroid-${state.id}`}
      texture={texture}
      x={state.pos.x}
      y={state.pos.y}
      anchor={0.5}
    />
  );
}
