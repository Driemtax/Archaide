import { PlayerInfo } from "@/types";
import { Avatar, AvatarImage, AvatarFallback } from "./ui/avatar";
import { Card, CardContent, CardFooter } from "./ui/card";
import { ReactNode } from "react";

interface UserDisplayProps {
  player: PlayerInfo;
  isYousrself: boolean;
  children?: ReactNode;
  score?: number;
}

export default function UserDisplay(props: UserDisplayProps) {
  const { player, isYousrself, children, score } = props;
  return (
    <Card>
      <CardContent className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-2">
        <div className="flex flex-col sm:flex-row justify-start sm:items-center items-start gap-2">
          <Avatar>
            <AvatarImage src={player.avatarUrl} />
            <AvatarFallback>{player.name.at(0)}</AvatarFallback>
          </Avatar>
          <span className="text-lg font-semibold font-arcade gap-4">
            <span>{player.name}</span>
            <span>{isYousrself ? " (You)" : ""}</span>
          </span>
        </div>
        <span className="text-sm font-medium leading-none font-arcade">
          Score: {score || player.score}
        </span>
      </CardContent>
      <CardFooter className="gap-3">{children}</CardFooter>
    </Card>
  );
}
