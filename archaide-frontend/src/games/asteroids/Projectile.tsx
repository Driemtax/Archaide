import { extend } from "@pixi/react";
import { Container, Graphics } from "pixi.js";
import { COLORS, SCREEN } from "./config";
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
        g.circle(0, 0, 3 * SCREEN.scaling_factor);
        g.fill();
      }}
      x={state.pos.x * SCREEN.scaling_factor}
      y={state.pos.y * SCREEN.scaling_factor}
    />
  );
}
