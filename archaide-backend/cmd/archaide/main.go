package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/Driemtax/Archaide/internal/coms"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", ":3030", "http service address")

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // TODO change to a more secure function...
}

func serveWs(hub *coms.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	log.Println("Client connected from:", conn.RemoteAddr())

	// Create a new client for a new connection
	client := &coms.Client{
		Hub:          hub,
		Conn:         conn,
		Send:         make(chan []byte, 256), // Buffer size for outgoing messages
		Id:           uuid.New().String(),
		Score:        0,
		SelectedGame: "",
	}
	client.Hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}

func main() {
	flag.Parse()
	hub := coms.NewHub()
	go hub.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "web/lobby.html")
	})

	log.Printf("Server starting on %s", *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
