import { useEffect, useRef, useState, useCallback } from "react";

interface WebSocketOptions {
  /** Callback function invoked when a message is received from the server. */
  onMessage: (event: MessageEvent<string>) => void; // Assuming messages are strings
  /** Callback function invoked when a connection error occurs or the connection closes unexpectedly. */
  onError?: (event: Event | CloseEvent) => void;
  /** Optional callback function invoked when the connection is successfully opened. */
  onOpen?: (event: Event) => void;
  /** Optional callback function invoked when the connection is closed (either cleanly or unexpectedly). */
  onClose?: (event: CloseEvent) => void;
}

/**
 * A very simple React hook for managing a single WebSocket connection.
 * Establishes a connection when a URL is provided and reports errors.
 * Does not handle automatic reconnection attempts.
 *
 * @param url The WebSocket server URL, or `null` to disconnect and prevent connection.
 * @param options Callback functions for various WebSocket events.
 */
export function useWebSocket(url: string | null, options: WebSocketOptions) {
  const { onMessage, onError, onOpen, onClose } = options;
  const ws = useRef<WebSocket | null>(null);
  const [error, setError] = useState<Event | CloseEvent | null>(null);
  const [readyState, setReadyState] = useState<number>(WebSocket.CLOSED);

  useEffect(() => {
    if (!url) {
      if (ws.current) {
        console.log("WebSocket: Closing connection because URL is null.");
        // 1000 means a normal closure
        ws.current.close(1000, "URL removed");
        ws.current = null;
      }
      setReadyState(WebSocket.CLOSED);
      setError(null);
      return;
    }

    // --- Connection Setup ---
    setError(null); // Reset error state on a new connection attempt
    setReadyState(WebSocket.CONNECTING);
    console.log(`WebSocket: Attempting to connect to ${url}...`);
    const socket = new WebSocket(url);
    ws.current = socket;

    // --- Event Handlers ---
    socket.onopen = (event) => {
      console.log("WebSocket: Connection opened.");
      setError(null);
      setReadyState(WebSocket.OPEN);
      onOpen?.(event);
    };

    socket.onmessage = (event: MessageEvent<string>) => {
      onMessage(event);
    };

    socket.onerror = (event) => {
      console.error("WebSocket: Error occurred.", event);
      setError(event);
      setReadyState(socket.readyState);
      onError?.(event);
    };

    socket.onclose = (event) => {
      console.log(
        `WebSocket: Connection closed (Code: ${event.code}, Reason: ${event.reason}, Clean: ${event.wasClean})`,
      );
      setReadyState(WebSocket.CLOSED);
      ws.current = null;

      // If the connection closed uncleanly and no specific 'error' event was already captured,
      // treat this unexpected closure as an error scenario.
      if (!event.wasClean && error === null) {
        console.warn("WebSocket: Connection closed unexpectedly.");
        // Create a generic event to represent the unexpected closure if onError expects an Event.
        const closeErrorEvent = new CustomEvent("websocketerror", {
          detail: event,
        });
        setError(closeErrorEvent); // Set the error state
        onError?.(closeErrorEvent); // Notify the user via the onError callback
      }

      onClose?.(event); // Call the user-provided onClose callback regardless of clean/unclean closure
    };

    // --- Cleanup Function ---
    // This function is executed when the component unmounts or when the dependencies
    // (like 'url') change, triggering the effect to run again. It ensures proper cleanup.
    return () => {
      if (socket) {
        console.log("WebSocket: Cleaning up connection.");
        // Remove event listeners to prevent memory leaks and potential issues during closing.
        socket.onopen = null;
        socket.onmessage = null;
        socket.onerror = null;
        socket.onclose = null;
        // Check if the socket is still in a state where closing is necessary.
        if (
          socket.readyState === WebSocket.OPEN ||
          socket.readyState === WebSocket.CONNECTING
        ) {
          socket.close(1000, "Component unmounted or URL changed");
        }
        ws.current = null; // Ensure the ref is cleared
        setReadyState(WebSocket.CLOSED); // Set state to closed
      }
    };
  }, [url, onMessage, onError, onOpen, onClose, error]);

  // Function to send messages over the WebSocket connection.
  const sendMessage = useCallback(
    (data: string | ArrayBufferLike | Blob | ArrayBufferView) => {
      // Only send if the current WebSocket instance exists and is in the OPEN state.
      if (ws.current?.readyState === WebSocket.OPEN) {
        ws.current.send(data);
      } else {
        console.warn("WebSocket is not connected. Cannot send message.");
      }
    },
    [],
  );

  return {
    /** Function to send data over the WebSocket connection. */
    sendMessage,
    /** The last encountered error (Event or CloseEvent for unexpected closures), or `null`. */
    error,
    /** The current connection state (WebSocket.CONNECTING, WebSocket.OPEN, WebSocket.CLOSING, WebSocket.CLOSED). */
    readyState,
  };
}
