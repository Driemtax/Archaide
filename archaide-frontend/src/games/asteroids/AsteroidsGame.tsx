import { Application, extend } from "@pixi/react";
import { Container, Graphics } from "pixi.js";
import * as PIXI from "pixi.js";
import { useWebSocketContext } from "../../hooks/useWebSocketContext";
import { ClientMessage } from "../../types";
import { useEffect, useState, useRef } from "react"; // Added useEffect, useState, useRef
import { COLORS, SCREEN } from "./config";
import AsteroidsStage from "./AsteroidsStage";
import UserDisplay from "@/components/UserDisplay";
import { Badge } from "@/components/ui/badge";

// Exactly the doubled value of the actual asteroid update rate
const SEND_INTERVAL = 1000 / 60;

extend({ Container, Graphics });

const ASTEROID_ASSET_PATHS = [
  "assets/sprite_large_asteroid0.png",
  "assets/sprite_large_asteroid1.png",
  "assets/sprite_large_asteroid2.png",
  "assets/sprite_middle_asteroid0.png",
  "assets/sprite_middle_asteroid1.png",
  "assets/sprite_middle_asteroid2.png",
  "assets/sprite_small_asteroid0.png",
  "assets/sprite_small_asteroid1.png",
  "assets/sprite_small_asteroid2.png",
  "assets/sprite_asteroids_player.png",
  "assets/sprite_asteroids_own_player.png",
];

export default function Asteroids() {
  const { sendMessage, asteroidState, players, myClientId } =
    useWebSocketContext();
  const [keysPressed, setKeysPressed] = useState({
    left: false,
    right: false,
    up: false,
    shoot: false,
  });
  const [assetsLoaded, setAssetsLoaded] = useState<boolean>(false);

  // Use a ref for keysPressed inside the interval to always get the latest state
  // without re-creating the interval on every key press.
  const keysPressedRef = useRef(keysPressed);
  useEffect(() => {
    keysPressedRef.current = keysPressed;
  }, [keysPressed]);

  useEffect(() => {
    const loadAssets = async () => {
      try {
        await PIXI.Assets.load(ASTEROID_ASSET_PATHS);
        console.log("Assets loaded!");
        setAssetsLoaded(true);
      } catch (error) {
        console.error("Error loading assets:", error);
      }
    };

    loadAssets();

    const handleKeyDown = (event: KeyboardEvent) => {
      switch (event.key) {
        case "ArrowLeft":
          if (!keysPressedRef.current.left) {
            setKeysPressed((prev) => ({ ...prev, left: true }));
          }
          break;
        case "ArrowRight":
          if (!keysPressedRef.current.right) {
            setKeysPressed((prev) => ({ ...prev, right: true }));
          }
          break;
        case "ArrowUp":
          if (!keysPressedRef.current.up) {
            setKeysPressed((prev) => ({ ...prev, up: true }));
          }
          break;
        case " ": // Space bar
          if (!keysPressedRef.current.shoot) {
            setKeysPressed((prev) => ({ ...prev, shoot: true }));
          }
          break;
      }
      // Prevent default for keys that might scroll the page
      if (["ArrowLeft", "ArrowRight", "ArrowUp", " "].includes(event.key)) {
        event.preventDefault();
      }
    };

    const handleKeyUp = (event: KeyboardEvent) => {
      switch (event.key) {
        case "ArrowLeft":
          if (keysPressedRef.current.left) {
            setKeysPressed((prev) => ({ ...prev, left: false }));
          }
          break;
        case "ArrowRight":
          if (keysPressedRef.current.right) {
            setKeysPressed((prev) => ({ ...prev, right: false }));
          }
          break;
        case "ArrowUp":
          if (keysPressedRef.current.up) {
            setKeysPressed((prev) => ({ ...prev, up: false }));
          }
          break;
        case " ": // Space bar
          if (keysPressedRef.current.shoot) {
            setKeysPressed((prev) => ({ ...prev, shoot: false }));
          }
          break;
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    window.addEventListener("keyup", handleKeyUp);

    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("keyup", handleKeyUp);
      PIXI.Assets.unload(ASTEROID_ASSET_PATHS);
    };
  }, []);

  useEffect(() => {
    const intervalId = setInterval(() => {
      const msg: ClientMessage = {
        type: "asteroids_input",
        payload: keysPressedRef.current, // Use the ref here
      };
      sendMessage(msg);
    }, SEND_INTERVAL);

    return () => {
      clearInterval(intervalId); // Cleanup interval on component unmount
    };
  }, [sendMessage]);

  if (!assetsLoaded) {
    return <p>Loading Assets ...</p>;
  }

  return (
    <div className="px-4 py-2 lg:px-8 lg:py-4">
      <h1 className="scroll-m-20 text-4xl font-extrabold tracking-tight lg:text-5xl font-arcade pb-4">
        Asteroids
      </h1>
      <div className="flex flex-col-reverse xl:flex-row gap-4">
        <div className="hidden 2xl:grid gap-4 grid-cols-1 w-1/3">
          {Object.entries(asteroidState?.players || {}).map(([, p]) => (
            <UserDisplay
              player={players?.[p.id]}
              isYousrself={myClientId === p.id}
              score={p.score}
            >
              {p.isInvincible && (
                <Badge className="font-arcade" variant="outline">
                  Has respawn protection
                </Badge>
              )}
              <div className="flex flex-row gap-2">
                {[...Array(p.health)].map(() => (
                  <img src="/assets/sprite_asteroids_player.png" />
                ))}
              </div>
            </UserDisplay>
          ))}
        </div>
        <div className="border shadow-sm rounded-xl p-1">
          <Application
            width={SCREEN.width * SCREEN.scaling_factor}
            height={SCREEN.height * SCREEN.scaling_factor}
            backgroundColor={COLORS.black}
            antialias={true}
          >
            <AsteroidsStage />
          </Application>
        </div>
      </div>
    </div>
  );
}
