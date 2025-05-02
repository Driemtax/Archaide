import { useEffect } from "react";
import { Application, extend } from "@pixi/react";
import { Container, Graphics} from "pixi.js";
import type { PongStatePayload } from "../../types";

interface PongGameProps {
  gameState: PongStatePayload;
  onMove: (direction: string) => void;
}

const PaddleWidth = 20;
const PaddleHeight = 100;
const BallRadius = 10;

extend({ Container, Graphics });

function PongStage({ gameState, onMove }: PongGameProps) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "ArrowUp") onMove("up");
      if (e.key === "ArrowDown") onMove("down");
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onMove]);

  console.log(gameState)
  return (
    <pixiContainer>
      {/* Paddle 1 */}
      <pixiGraphics
        draw={(g: Graphics) => {
          g.clear();
          g.fill(0xffffff);
          g.rect(0, 0, PaddleWidth, PaddleHeight);
          g.fill()
        }}
        x={0}
        y={gameState.Paddle1Y - (PaddleHeight / 2)}
      />
      {/* Paddle 2 */}
      <pixiGraphics
        draw={(g: Graphics) => {
          g.fill(0xffffff);
          g.rect(0, 0, PaddleWidth, PaddleHeight);
          g.fill();
        }}
        x={800 - PaddleWidth}
        y={gameState.Paddle2Y - (PaddleHeight / 2)}
      />
      {/* Ball */}
      <pixiGraphics
        draw={(g: Graphics) => {
          g.fill(0xffffff);
          g.circle(0, 0, BallRadius);
          g.fill();
        }}
        x={gameState.BallX}
        y={gameState.BallY}
      />
    </pixiContainer>
  );
}

export default function PongGame(props: PongGameProps) {
  return (
    <div style={{border: "1px solid white"}}>
    <Application width={800} height={600} backgroundColor={0x000000} antialias>
      <PongStage {...props} />
    </Application>
    </div>
  );
}
