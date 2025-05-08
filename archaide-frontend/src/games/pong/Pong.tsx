import { useEffect } from "react";
import { Application, extend } from "@pixi/react";
import { Container, Graphics } from "pixi.js";
import type {
  ClientMessage,
  PongPlayerMove,
  PongStatePayload,
} from "../../types";
import { useWebSocketContext } from "../../hooks/useWebSocketContext";

interface PongGameProps {
  clientID: string;
  gameState: PongStatePayload;
  onMove: (direction: PongPlayerMove) => void;
}

interface HudProps {
  player1Score: number;
  player2Score: number;
}

const PaddleWidth = 20;
const PaddleHeight = 100;
const BallRadius = 10;

// const COUNTDOWN_START = 3;

const BG_COLOR = 0x181818;
const PADDLE_COLOR_1 = 0x000000;
const PADDLE_COLOR_2 = 0xcccccc;
const BALL_COLOR = 0xd4ffd4;

extend({ Container, Graphics });

function GameHUD({ player1Score, player2Score }: HudProps) {
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
      {/* {countdown > 0 && (
        <div style={{ fontSize: 48, marginBottom: 8 }}>
          Spiel startet in {countdown}...
        </div>
      )} */}
    </div>
  );
}

function PongStage({ clientID, gameState, onMove }: PongGameProps) {
  let player = 0
  if (gameState.player_1 === clientID) {
    player = 1
  } else if (gameState.player_2 === clientID) {
    player = 2
  }

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      console.log("Key pressed:", e.key);
      if (e.key === "ArrowUp") onMove("up");
      if (e.key === "ArrowDown") onMove("down");
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onMove]);

  const pixiContainer = (
    <pixiContainer>
      {/* Paddle 1 */}
      <pixiGraphics
        draw={(g: Graphics) => {
          g.clear();
          g.fill(player === 1 ? PADDLE_COLOR_2 : PADDLE_COLOR_1);
          g.setStrokeStyle({color: 0xffffff, width: 2})
          g.rect(0, 0, PaddleWidth, PaddleHeight);
          g.fill();
          g.stroke();
        }}
        x={0}
        y={gameState.paddle_1_y - PaddleHeight / 2}
      />

      {/* Paddle 2 */}
      <pixiGraphics
        draw={(g: Graphics) => {
          g.fill(player === 1 ? PADDLE_COLOR_1 : PADDLE_COLOR_2);
          g.rect(0, 0, PaddleWidth, PaddleHeight);
          g.fill();
          if (player === 1) {
            g.setStrokeStyle({color: 0xffffff, width: 2})
            g.stroke();
          }
        }}
        x={800 - PaddleWidth}
        y={gameState.paddle_2_y - PaddleHeight / 2}
      />
      {/* Ball */}
      <pixiGraphics
        draw={(g: Graphics) => {
          g.fill(BALL_COLOR);
          g.circle(0, 0, BallRadius);
          g.fill();
        }}
        x={gameState.ball_x}
        y={gameState.ball_y}
      />
    </pixiContainer>
  );

  return pixiContainer;
}

export default function PongGame() {
  const { myClientId, pongState, sendMessage } = useWebSocketContext();
  //const [countdown, setCountdown] = useState(COUNTDOWN_START);

  // useEffect(() => {
  //   if (countdown === 0) return;

  //   const timer = setInterval(() => {
  //     setCountdown((prev) => prev > 0 ? prev -1 : 0);
  //   }, 1000);

  //   return () => clearInterval(timer)

  // })

  const sendMove = (dir: PongPlayerMove) => {
    const msg: ClientMessage = {
      type: "pong_input",
      payload: {
        direction: dir,
      },
    };

    sendMessage(msg);
  };

  if (!pongState) {
    // Waiting for the first game state ensuring
    // that a game state is always present
    return <p>Loading game...</p>;
  }

  return (
    <div style={{ width: 802, margin: "0 auto" }}>
      <GameHUD
        player1Score={pongState?.score_1 || 0}
        player2Score={pongState?.score_2 || 0}
      />
      <div style={{ border: "1px solid white" }}>
        <Application
          width={800}
          height={600}
          backgroundColor={BG_COLOR}
          antialias
        >
          <PongStage
            clientID={myClientId}
            onMove={(dir: PongPlayerMove) => sendMove(dir)}
            gameState={pongState}
          />
        </Application>
      </div>
    </div>
  );
}
