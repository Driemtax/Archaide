import { ClientMessage } from "../types";
import { useWebSocketContext } from "../hooks/useWebSocketContext";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import UserDisplay from "@/components/UserDisplay";

export default function LobbyView() {
  const { readyState, myClientId, players, availableGames, sendMessage } =
    useWebSocketContext();

  const handleSelectGame = (gameName: string) => {
    if (Object.keys(players).length < 2) {
      toast(
        "❌ There have to be at least two players in the lobby to start a game",
      );
      return;
    }
    if (readyState !== WebSocket.OPEN) {
      toast("Not connected to server.");
      return;
    }
    const message: ClientMessage = {
      type: "select_game",
      payload: { game: gameName },
    };
    sendMessage(message);
  };

  return (
    <div className="px-4 py-2 lg:px-8 lg:py-4 max-h-screen">
      <h1 className="text-3xl font-arcade font-bold pb-8">Archaide</h1>

      <div className="flex flex-col-reverse lg:flex-row justify-center items-center lg:items-start gap-8 overflow-scroll">
        <div className="w-full lg:w-1/3">
          <h2 className="text-base font-arcade pb-2">Players</h2>
          {Object.keys(players).length > 0 ? (
            <div className="grid gap-4 columns-1">
              {Object.entries(players).map(([clientId, playerInfo]) => (
                <UserDisplay
                  key={clientId}
                  player={playerInfo}
                  isYousrself={clientId === myClientId}
                >
                  <Badge className="font-arcade" variant="outline">
                    {playerInfo.inGame ? "Is currently playing" : "Is in Lobby"}
                  </Badge>
                  {playerInfo.selectedGame && (
                    <Badge variant="outline" className="font-arcade">
                      Selected {playerInfo.selectedGame}
                    </Badge>
                  )}
                </UserDisplay>
              ))}
            </div>
          ) : (
            <p>No other players currently in the lobby.</p>
          )}
        </div>

        <div className="w-full lg:w-2/3">
          <h2 className="text-base font-arcade pb-2">Games</h2>
          {availableGames.length > 0 ? (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              {availableGames.map((game) => (
                <Card key={game.name}>
                  <CardHeader>
                    <CardTitle className="font-arcade">{game.name}</CardTitle>
                    <CardDescription className="font-arcade">
                      {game.description}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <Button
                      onClick={() => handleSelectGame(game.name)}
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
      </div>
    </div>
  );
}
