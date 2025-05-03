import { JSX, useState, useEffect, useRef, useCallback } from "react";

import type {
  ServerMessage,
  WelcomePayload,
  UpdateLobbyPayload,
  GameSelectedPayload,
  ErrorPayload,
  ClientSelectGameMessage,
  ClientSelectGamePayload,
  PongStatePayload
} from "../types";
import "./Lobby.css";
import PongGame from "../games/pong/Pong";

// Simple basic lobby component
// We should iterate over it to make it a lot more attractive
// I think we should also switch from using css to tailwind...
// Just for convenience and being able to use shadcdn
function Lobby(): JSX.Element {
  const [connectionStatus, setConnectionStatus] =
    useState<string>("Connecting...");
  const [logMessages, setLogMessages] = useState<string[]>([]);
  const [players, setPlayers] = useState<Record<string, number>>({});
  const [availableGames, setAvailableGames] = useState<string[]>([]);
  const [myClientId, setMyClientId] = useState<string>("");
  const [selectedGame, setSelectedGame] = useState<string>("");
  const [selectedGameInfo, setSelectedGameInfo] = useState<string>("");
  const [isGameSelectionDisabled, setIsGameSelectionDisabled] =
    useState<boolean>(false);

  const ws = useRef<WebSocket | null>(null);
  const logDivRef = useRef<HTMLDivElement | null>(null);

  // Game states
  const [pongState, setPongState] = useState<PongStatePayload>({
    BallX: 400,
    BallY: 300,
    Paddle1Y: 300,
    Paddle2Y: 300,
    Score1: 0,
    Score2: 0
  });

  // --- WebSocket Logic ---

  const logMessage = useCallback((message: string): void => {
    setLogMessages((prevLogs) => [...prevLogs, message]);
  }, []);

  const handleMessage = useCallback(
    (messageData: string): void => {
      let message: ServerMessage;
      try {
        message = JSON.parse(messageData) as ServerMessage;

        logMessage(`<- Received: ${JSON.stringify(message, null, 2)}`);

        switch (message.type) {
          case "welcome": {
            const payload = message.payload as WelcomePayload;
            setMyClientId(payload.clientId);
            setAvailableGames(payload.currentGames ?? []);
            setConnectionStatus(`Connected as ${payload.clientId}`);
            setIsGameSelectionDisabled(false);
            setSelectedGameInfo("");
            break;
          }
          case "update_lobby": {
            const payload = message.payload as UpdateLobbyPayload;
            setPlayers(payload.players ?? {});
            break;
          }
          case "game_selected": {
            const payload = message.payload as GameSelectedPayload;
            setSelectedGameInfo(
              `Game selected: ${payload.selectedGame}! Waiting for game start...`,
            );
            setSelectedGame(payload.selectedGame)
            setIsGameSelectionDisabled(true);
            break;
          }
          case "error": {
            const payload = message.payload as ErrorPayload;
            const errorMsg = `Server Error: ${payload.message}`;
            logMessage(errorMsg);
            alert(errorMsg);
            break;
          }

          case "pong_state": {
            const payload = message.payload as PongStatePayload;
            setPongState(payload);
            break;
          }
          default:
            logMessage(`Unknown message type received: ${message.type}`);
        }
      } catch (e) {
        logMessage(
          `Error parsing message: ${e instanceof Error ? e.message : String(e)} - Raw data: ${messageData}`,
        );
        console.error("Error parsing WebSocket message:", e);
      }
    },
    [logMessage],
  );

  const connect = useCallback((): void => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      logMessage("Closing existing WebSocket connection before reconnecting.");
      ws.current.onclose = null;
      ws.current.close();
    }
    if (ws.current && ws.current.readyState === WebSocket.CONNECTING) {
      logMessage("Connection attempt already in progress.");
      return;
    }

    const proto = "ws:";
    const host = "localhost:3030";
    const wsUrl = `${proto}//${host}/ws`;

    logMessage(`Attempting to connect to ${wsUrl}...`);
    setConnectionStatus("Connecting...");

    ws.current = new WebSocket(wsUrl);

    ws.current.onopen = () => {
      logMessage("WebSocket connection opened.");
    };

    ws.current.onmessage = (event: MessageEvent<string>) => {
      handleMessage(event.data);
    };

    ws.current.onclose = (event: CloseEvent) => {
      if (event.code !== 1000 && event.code !== 1001) {
        setConnectionStatus("Disconnected. Attempting to reconnect...");
        logMessage(
          `WebSocket connection closed unexpectedly (Code: ${event.code}, Reason: ${event.reason || "No reason given"}). Reconnecting in 3 seconds...`,
        );
        ws.current = null;
        setMyClientId("");
        setPlayers({});
        setAvailableGames([]);
        setSelectedGameInfo("");
        setIsGameSelectionDisabled(false);
        setTimeout(connect, 3000);
      } else {
        setConnectionStatus("Disconnected.");
        logMessage(
          `WebSocket connection closed normally (Code: ${event.code}, Reason: ${event.reason || "Normal"}).`,
        );
        ws.current = null;
        setMyClientId("");
        setPlayers({});
        setAvailableGames([]);
        setSelectedGameInfo("");
        setIsGameSelectionDisabled(false);
      }
    };

    ws.current.onerror = (event: Event) => {
      setConnectionStatus("Connection Error");
      logMessage("WebSocket error occurred. See browser console for details.");
      console.error("WebSocket Error:", event);
    };
  }, [logMessage, handleMessage]); // Depends on memoized handlers

  // --- Effects ---

  useEffect(() => {
    connect();

    return () => {
      if (ws.current) {
        logMessage("Closing WebSocket connection on component unmount.");
        ws.current.onclose = null;
        ws.current.close(1000, "Client component unmounted");
        ws.current = null;
      }
    };
  }, [connect, logMessage]);

  useEffect(() => {
    if (logDivRef.current) {
      logDivRef.current.scrollTop = logDivRef.current.scrollHeight;
    }
  }, [logMessages]);

  // --- Event Handlers ---

  const handleSelectGame = (gameName: string): void => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      logMessage("Cannot select game: Not connected.");
      alert("Connection lost. Please wait for reconnection.");
      return;
    }

    const payload: ClientSelectGamePayload = { game: gameName };
    const message: ClientSelectGameMessage = {
      type: "select_game",
      payload: payload,
    };
    const messageString = JSON.stringify(message);

    logMessage(`-> Sending: ${messageString}`);
    ws.current.send(messageString);

    setSelectedGameInfo(
      `You selected: ${gameName}. Waiting for server confirmation...`,
    );
    setIsGameSelectionDisabled(true);
  };

  // --- Rendering  ---

  return (
    <div className="page-container">
      <div className="game-canvas-container">
      {selectedGame === 'Pong' && (
        <PongGame 
          gameState={pongState}
          onMove={(direction) => {
            ws.current?.send(JSON.stringify({
              type: "pong_input",
              payload: { direction }
            }))
          }}/>
      )}
      </div>
      {selectedGame === "" && (
        <div className="lobby-container">
        <h1>Archaide Lobby</h1>

        <div className="log-container" ref={logDivRef}>
          {logMessages.map((msg, index) => (
            <p key={index}>{msg}</p>
          ))}
        </div>

        <div className="status-container">Status: {connectionStatus}</div>

        <div className="main-content">
          <div className="players-container">
            <h2>Players in Lobby</h2>
            {Object.keys(players).length > 0 ? (
              Object.entries(players).map(([clientId, score]) => (
                <div key={clientId} className="player">
                  <strong>
                    {clientId === myClientId ? `${clientId} (You)` : clientId}:
                  </strong>{" "}
                  {score} points
                </div>
              ))
            ) : (
              <p>No other players currently in the lobby.</p>
            )}
          </div>

          <div className="games-container">
            <h2>Select a Game</h2>
            {availableGames.length > 0 ? (
              availableGames.map((game) => (
                <button
                  key={game}
                  onClick={() => handleSelectGame(game)}
                  disabled={isGameSelectionDisabled}
                >
                  {game}
                </button>
              ))
            ) : (
              <p>No games available to join right now.</p>
            )}onMove("up")
          </div>
        </div>
      </div>
      )}
    </div>
  );
}

export default Lobby;
