import { useEffect, useState } from "react";
import { ClientMessage } from "../types";
import { useWebSocketContext } from "../hooks/useWebSocketContext";

export default function LobbyView() {
  const {
    readyState,
    myClientId,
    players,
    availableGames,
    sendMessage,
    gameError,
  } = useWebSocketContext();

  const [isGameSelectionPending, setIsGameSelectionPending] = useState(false);

  const handleSelectGame = (gameName: string) => {
    if (readyState !== WebSocket.OPEN) {
      // TODO implement some kind of toast system or notification system...
      alert("Not connected to server.");
      return;
    }
    setIsGameSelectionPending(true);
    const message: ClientMessage = {
      type: "select_game",
      payload: { game: gameName },
    };
    sendMessage(message);
  };

  // If the component is newly rendered it came back because the
  // player went back to the lobby so we always reset the game selection state
  // to false.
  useEffect(() => {
    setIsGameSelectionPending(false);
  }, []);

  return (
    <div>
      <h1>Archaide Lobby</h1>

      {/* Status Container  */}
      <h3>
        Status: {readyState}{" "}
        {readyState === WebSocket.OPEN && `(ID: ${myClientId})`}
        {gameError && <p> Error: {gameError}</p>}
      </h3>

      <h2>Players in Lobby</h2>
      {Object.keys(players).length > 0 ? (
        <ul>
          {Object.entries(players).map(([clientId, playerInfo]) => (
            <li key={clientId}>
              <strong>
                {clientId === myClientId ? `${clientId} (You)` : clientId}:
              </strong>{" "}
              Score: {playerInfo.score} points
            </li>
          ))}
        </ul>
      ) : (
        <p>No other players currently in the lobby.</p>
      )}

      <h2>Select a Game</h2>
      {availableGames.length > 0 ? (
        <ul>
          {availableGames.map((game) => (
            <li key={game}>
              <button
                onClick={() => handleSelectGame(game)}
                disabled={
                  isGameSelectionPending || readyState !== WebSocket.OPEN
                }
              >
                {game}
              </button>
            </li>
          ))}
        </ul>
      ) : (
        <p>No games available right now.</p>
      )}
    </div>
  );
}
