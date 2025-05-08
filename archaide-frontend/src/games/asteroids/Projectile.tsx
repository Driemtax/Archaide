import { extend } from "@pixi/react";
import { Container, Graphics } from "pixi.js";
import { COLORS } from "./config";
import { AsteroidsProjectileState } from "../../types";

extend({ Container, Graphics });

interface ProjectileProps {
  state: AsteroidsProjectileState;
}

export default function Projectile(props: ProjectileProps) {
  const { state } = props;

  return (
    <pixiGraphics
      key={`projectile-${state.id}`}
      draw={(g) => {
        g.clear();
        g.fill(COLORS.white);
        g.circle(0, 0, 3);
        g.fill();
      }}
      x={state.pos.x}
      y={state.pos.y}
    />
  );
}
