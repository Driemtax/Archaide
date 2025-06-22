## 👾 Archaide 🎮

Ready for a showdown? Archaide is your go-to multiplayer arcade platform built for epic battles with your friends! ⚔️ Challenge your pals and relive the classic arcade fun. 🕹️

This project was built with a focus on creating a robust, real-time, and concurrent backend system from the ground up to power a seamless multiplayer experience.
## ✨ Features & Technical Highlights

This project is more than just a game; it's an exploration of modern backend and frontend technologies for real-time applications. Here are some of the key architectural and technical features we're proud of:
#### 1. Server-Authoritative Architecture with Go

The entire game logic is managed by the backend, which is built in Go. This server-authoritative model is a deliberate architectural choice to ensure fairness and prevent cheating, as the server is the single source of truth for all game states.

    Why Go? We chose Go for its exceptional support for concurrency through Goroutines and Channels. This is crucial for a multiplayer server that needs to handle numerous simultaneous client connections, game states, and events efficiently.

#### 2. Real-time Communication via WebSockets

At the heart of Archaide is a real-time communication layer built on WebSockets. We designed a custom messaging protocol to ensure a lean and efficient data flow between the server and the clients.

    Custom Message Protocol (message.go): To standardize communication, we defined a clear set of message types (e.g., welcome, update_lobby, game_state). Each message has a type and a flexible JSON payload, allowing the frontend to react dynamically to different server events.
    Go

    // A snippet from internal/message/message.go
    type Message struct {
        Type    MessageType     `json:"type"`
        Payload json.RawMessage `json:"payload"` 
    }

    const (
        Welcome     MessageType = "welcome"
        UpdateLobby MessageType = "update_lobby"
        SelectGame  MessageType = "select_game"
        // ... and game-specific types
    )

#### 3. Concurrent Lobby & Game Management (The Hub)

The most significant technical challenge and achievement of this project is the management of concurrent operations. The central component, hub.go, acts as the nervous system of the server.

Goroutines & Channels: The Hub uses channels (`Register`, `unregister`, `incoming`) to handle client events asynchronously without blocking the main loop. Each client connection runs in its own Goroutine, enabling massive scalability.

State Protection with Mutexes: To prevent race conditions and ensure data consistency across multiple Goroutines, we meticulously manage access to shared state (like clients, activeGames). We use `sync.RWMutex` to protect critical data structures from concurrent access. Mastering this was a key learning experience, turning hours of debugging potential deadlocks into a robust and stable system.
    
```Go

    // A snippet from internal/hub/hub.go demonstrating state protection
    type Hub struct {
        // ...
        clients      map[*Client]bool
        activeGames  map[string]game.Game
        clientToGame map[*Client]string
        gameMutex    sync.RWMutex // The guardian of our shared state
    }

    func (h *Hub) Run() {
        for {
            select {
            case client := <-h.Register:
                h.gameMutex.Lock()
                h.clients[client] = true
                h.gameMutex.Unlock()
                // ...
```

#### 4. System Robustness & Graceful Disconnects

We built the system to be resilient. If a player disconnects unexpectedly (e.g., closes their browser), the server immediately detects this. The Hub ensures the player is removed from their current lobby or active game, and all other players are notified via a lobby update. This prevents games from stalling and maintains a smooth user experience for everyone else.

#### 5. Dynamic Frontend with React & PixiJS

The frontend is a Single Page Application (SPA) built with React and PixiJS.

Why this stack? We chose React for its component-based architecture and robust ecosystem, which allowed us to quickly build a responsive user interface. For rendering the actual gameplay within an HTML canvas, we integrated PixiJS, a powerful 2D rendering engine that provides the performance needed for smooth animations and an arcade feel.

## 🚀 Getting Started: Setup Guide ⚙️

Follow these steps to get Archaide up and running on your local machine. You'll need both the backend server and the frontend application running simultaneously.
#### 1. Backend Server Setup (Go) 🖥️

The backend powers the core game logic. Let's get it started!

Navigate to the backend project directory:
    
    ```Bash 
    cd archaide-backend```

Start the server using one of the following commands:

Option A: Using Make (if available)
```Bash
# This command handles building and running the server
make start
```

Option B: Using Go Run (if Make is not installed)
```Bash
# This command compiles and runs the main application file
go run cmd/archaide/main.go#
```

✅ Success! The backend server should now be running and listening on http://localhost:3030.

#### 2. Frontend Application Setup (React & PixiJS) 🎨 ✨

Now let's get the user interface running so you can see the action!

Navigate to the frontend project directory (in a new terminal window):
```Bash
cd archaide-frontend
```

Install the necessary project dependencies:
```Bash
npm install
```

Start the frontend development server:
```Bash
    npm run dev
```

✅ All Set! The frontend application should now be accessible in your web browser. Open up http://localhost:8080 🌐 to start playing!

## 🛠️ Future Enhancements

While we're proud of the current state of Archaide, there are some interessting topics that could be tackled in a bigger project scope:

Client-Side Prediction: To further enhance the user experience and make gameplay feel instantaneous even with network latency, implementing client-side prediction would be the next logical step.
More Games: The modular structure of the game logic is designed to be extensible. Adding more classic arcade games to the platform is a simply lovely joy.
Persistent Player Accounts & Leaderboards: Introducing user accounts and a global leaderboard to foster a more competitive community.
