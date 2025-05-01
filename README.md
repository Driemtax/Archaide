# 👾 Archaide 🎮

**Ready for a showdown?** Archaide is your go-to **multiplayer arcade platform** built for epic battles with your friends! ⚔️ Challenge your pals and relive the classic arcade fun. 🕹️

---

## 🚀 Getting Started: Setup Guide ⚙️

Follow these steps to get Archaide up and running on your local machine. You'll need both the backend server and the frontend application running simultaneously.

### 1. Backend Server Setup 🖥️ <0xF0><0x9F><0xA7><0xAE>

The backend powers the core game logic. Let's get it started!

1.  **Navigate** to the backend project directory:
    ```bash
    cd archaide-backend
    ```

2.  **Start the server** using *one* of the following commands:

    *   **Option A: Using Make (if available)**
        ```bash
        # This command handles building and running the server
        make start
        ```
    *   **Option B: Using Go Run (if Make is not installed)**
        ```bash
        # This command compiles and runs the main application file
        go run cmd/archaide/main.go
        ```

3.  ✅ **Success!** The backend server should now be running and listening on `http://localhost:3030`. You should see log output in your terminal.

---

### 2. Frontend Application Setup 🎨 ✨

Now let's get the user interface running so you can see the action!

1.  **Navigate** to the frontend project directory (in a *new* terminal window or tab):
    ```bash
    cd archaide-frontend
    ```

2.  **Install** the necessary project dependencies (you only need to do this the first time or when dependencies change):
    ```bash
    # This downloads all the required libraries for the frontend
    npm install
    ```

3.  **Start** the frontend development server:
    ```bash
    # This command usually starts a local web server with hot-reloading
    npm run dev
    ```

4.  ✅ **All Set!** The frontend application should now be accessible in your web browser. Open up [`http://localhost:8080`](http://localhost:8080) 🌐 to start playing!
