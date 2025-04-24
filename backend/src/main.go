package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", ":8080", "http service address")

// upgrader konvertiert HTTP-Verbindungen zu WebSockets.
// CheckOrigin wird hier unsicher konfiguriert, um alle Ursprünge zu erlauben (für Entwicklung).
// In Produktion solltest du dies einschränken!
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// serveWs behandelt WebSocket-Anfragen vom Peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	log.Println("Client connected from:", conn.RemoteAddr())

	// Erstelle einen neuen Client für diese Verbindung
	client := &Client{
		hub:          hub,
		conn:         conn,
		send:         make(chan []byte, 256), // Puffergröße für ausgehende Nachrichten
		id:           uuid.New().String(),    // Eindeutige ID generieren
		score:        0,                      // Start-Score
		selectedGame: "",                     // Noch kein Spiel ausgewählt
	}
	client.hub.register <- client // Registriere den Client beim Hub

	// Starte die Pump-Goroutinen für diesen Client
	// Diese Goroutinen laufen, bis die Verbindung geschlossen wird oder ein Fehler auftritt.
	go client.writePump()
	go client.readPump()

	// readPump kümmert sich darum, den Client beim Hub abzumelden, wenn die Verbindung endet.
}

func main() {
	flag.Parse()
	hub := newHub() // Erstelle den zentralen Hub
	go hub.run()    // Starte den Hub in einer eigenen Goroutine

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r) // Behandle WebSocket-Verbindungen
	})

	// Optional: Füge einen einfachen HTTP-Handler hinzu, um eine Test-HTML-Seite auszuliefern
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "lobby.html") // Annahme: Es gibt eine lobby.html im selben Verzeichnis
	})

	log.Printf("Server starting on %s", *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
