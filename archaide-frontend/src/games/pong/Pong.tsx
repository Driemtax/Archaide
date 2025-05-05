import { useEffect, useState } from "react";
import { Application, extend } from "@pixi/react";
import { Container, Graphics, Text } from "pixi.js";
import type { PongStatePayload } from "../../types";

interface PongGameProps {
  gameState: PongStatePayload;
  onMove: (direction: string) => void;
}

interface HudProps {
  player1Score: number;
  player2Score: number;
  countdown: number;
}

const PaddleWidth = 20;
const PaddleHeight = 100;
const BallRadius = 10;

const COUNTDOWN_START = 3;

const BG_COLOR = 0x181818;
const PADDLE_COLOR = 0xcccccc;
const BALL_COLOR = 0xd4ffd4;

extend({ Container, Graphics, Text });

function GameHUD({ player1Score, player2Score, countdown }: HudProps) {
  return (
    <div
      style={{
        width: 800,
        margin: "0 auto",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        color: "white",
        fontFamily: "Arial, sans-serif",
        userSelect: "none",
      }}
    >
      <h1 style={{ margin: "16px 0 8px 0", fontSize: 32 }}>PONG 2025</h1>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          width: "60%",
          fontSize: 28,
          marginBottom: 12,
        }}
      >
        <span>Spieler 1: {player1Score}</span>
        <span>Spieler 2: {player2Score}</span>
      </div>
      {countdown > 0 && (
        <div style={{ fontSize: 48, marginBottom: 8 }}>
          Spiel startet in {countdown}...
        </div>
      )}
    </div>
  );
}

function PongStage({ gameState, onMove }: PongGameProps) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "ArrowUp") onMove("up");
      if (e.key === "ArrowDown") onMove("down");
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onMove]);

  return (
    <pixiContainer>
      {/* Paddle 1 */}
      <pixiGraphics
        draw={(g: Graphics) => {
          g.clear();
          g.fill(PADDLE_COLOR);
          g.rect(0, 0, PaddleWidth, PaddleHeight);
          g.fill();
        }}
        x={0}
        y={gameState.Paddle1Y - PaddleHeight / 2}
      />
      {/* Paddle 2 */}
      <pixiGraphics
        draw={(g: Graphics) => {
          g.fill(PADDLE_COLOR);
          g.rect(0, 0, PaddleWidth, PaddleHeight);
          g.fill();
        }}
        x={800 - PaddleWidth}
        y={gameState.Paddle2Y - PaddleHeight / 2}
      />
      {/* Ball */}
      <pixiGraphics
        draw={(g: Graphics) => {
          g.fill(BALL_COLOR);
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
  const [countdown, setCountdown] = useState(COUNTDOWN_START);

  useEffect(() => {
    if (countdown === 0) return;

    const timer = setInterval(() => {
      setCountdown((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);

    return () => clearInterval(timer);
  });
  return (
    <div style={{ width: 802, margin: "0 auto" }}>
      <GameHUD
        player1Score={props.gameState.Score1}
        player2Score={props.gameState.Score2}
        countdown={countdown}
      />
      <div style={{ border: "1px solid white" }}>
        <Application
          width={800}
          height={600}
          backgroundColor={BG_COLOR}
          antialias
        >
          <PongStage {...props} />
        </Application>
      </div>
    </div>
  );
}
