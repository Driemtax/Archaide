import { Application, extend } from "@pixi/react";
import { Container, Graphics } from "pixi.js";
import * as PIXI from "pixi.js";
import { useWebSocketContext } from "../../hooks/useWebSocketContext";
import { ClientMessage } from "../../types";
import { useEffect, useState, useRef } from "react"; // Added useEffect, useState, useRef
import { COLORS, SCREEN } from "./config";
import AsteroidsStage from "./AsteroidsStage";

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
];

export default function Asteroids() {
  const { sendMessage } = useWebSocketContext();
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
    <div>
      <h1>Asteroids</h1>
      <Application
        width={SCREEN.width * SCREEN.scaling_factor}
        height={SCREEN.height * SCREEN.scaling_factor}
        backgroundColor={COLORS.black}
        antialias={true}
      >
        <AsteroidsStage />
      </Application>
    </div>
  );
}
