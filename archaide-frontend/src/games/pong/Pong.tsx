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
          g.fill(0x000000);
          g.rect(0, 0, PaddleWidth, PaddleHeight);
          g.fill()
        }}
        x={50}
        y={gameState.Paddle1Y}
      />
      {/* Paddle 2 */}
      <pixiGraphics
        draw={(g: Graphics) => {
          g.fill(0x000000);
          g.rect(0, 0, PaddleWidth, PaddleHeight);
          g.fill();
        }}
        x={800 - 50 - PaddleWidth}
        y={gameState.Paddle2Y}
      />
      {/* Ball */}
      <pixiGraphics
        draw={(g: Graphics) => {
          g.fill(0xff0000);
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
    <Application width={800} height={600} backgroundColor={0xffffff} antialias>
      <PongStage {...props} />
    </Application>
  );
}
