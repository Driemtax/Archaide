import { ClientMessage } from "../types";
import { useWebSocketContext } from "../hooks/useWebSocketContext";
import { Button } from "@/components/ui/button";
import { AvatarFallback, AvatarImage, Avatar } from "@/components/ui/avatar";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export default function LobbyView() {
  const { readyState, myClientId, players, availableGames, sendMessage } =
    useWebSocketContext();

  const handleSelectGame = (gameName: string) => {
    // TODO display to the client an error message if not enough players are inside of the lobby
    // BTW the server should also send an error message...
    // Currently the server just moves the client back to the lobby
    if (readyState !== WebSocket.OPEN) {
      // TODO implement some kind of toast system or notification system...
      alert("Not connected to server.");
      return;
    }
    const message: ClientMessage = {
      type: "select_game",
      payload: { game: gameName },
    };
    sendMessage(message);
  };

  return (
    <div>
      <h1 className="text-3xl underline font-bold">Archaide Lobby</h1>

      <h2>Players in Lobby</h2>
      {Object.keys(players).length > 0 ? (
        <ul>
          {Object.entries(players).map(([clientId, playerInfo]) => (
            <li key={clientId}>
              <Avatar>
                <AvatarImage src="https://avatar.iran.liara.run/public" />
                <AvatarFallback>CN</AvatarFallback>
              </Avatar>
              <strong>
                {clientId === myClientId ? `${clientId} (You)` : clientId}:
              </strong>
              <ul>
                <li>Score: {playerInfo.score} points</li>
                <li>In Game: {playerInfo.inGame ? "true" : "false"}</li>
                <li>
                  Selected Game: {playerInfo.selectedGame || "No game selected"}
                </li>
              </ul>
            </li>
          ))}
        </ul>
      ) : (
        <p>No other players currently in the lobby.</p>
      )}

      <h2>Select a Game</h2>
      {availableGames.length > 0 ? (
        <div className="grid grid-cols-4 gap-4">
          {availableGames.map((game) => (
            <Card key={game}>
              <CardHeader>
                <CardTitle>{game}</CardTitle>
                <CardDescription>
                  Select the game if you want to play it
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Button
                  onClick={() => handleSelectGame(game)}
                  disabled={readyState !== WebSocket.OPEN}
                >
                  Select
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <p>No games available right now.</p>
      )}
    </div>
  );
}
