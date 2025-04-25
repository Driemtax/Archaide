package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/Driemtax/Archaide/internal/coms"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", ":8080", "http service address")

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // TODO change to a more secure function...
}

// serveWs behandelt WebSocket-Anfragen vom Peer.
func serveWs(hub *coms.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	log.Println("Client connected from:", conn.RemoteAddr())

	// Erstelle einen neuen Client für diese Verbindung
	client := &coms.Client{
		Hub:          hub,
		Conn:         conn,
		Send:         make(chan []byte, 256), // Buffer size for outgoing messages
		Id:           uuid.New().String(),
		Score:        0,
		SelectedGame: "",
	}
	client.Hub.Register <- client

	// Starte die Pump-Goroutinen für diesen Client
	// Diese Goroutinen laufen, bis die Verbindung geschlossen wird oder ein Fehler auftritt.
	go client.WritePump()
	go client.ReadPump()

	// readPump kümmert sich darum, den Client beim Hub abzumelden, wenn die Verbindung endet.
}

func main() {
	flag.Parse()
	hub := coms.NewHub() // Erstelle den zentralen Hub
	go hub.Run()         // Starte den Hub in einer eigenen Goroutine

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r) // Behandle WebSocket-Verbindungen
	})

	// Optional: Füge einen einfachen HTTP-Handler hinzu, um eine Test-HTML-Seite auszuliefern
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
