package server

import (
	"log"
	"net/http"

	"github.com/Driemtax/Archaide/internal/hub"
)

func Run(addr *string) {
	hubInstance := hub.NewHub()

	go hubInstance.Run()

	// Register the WebSocket handler
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Pass the single hub instance to the handler
		serveWs(hubInstance, w, r)
	})

	// Simple handler for the root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Game server running. Connect via WebSocket on /ws"))
	})

	log.Printf("Server starting on %s", *addr)
	// Start the HTTP server
	err := http.ListenAndServe(*addr, nil) // Use the default ServeMux
	if err != nil {
		log.Fatalf("ListenAndServe failed: %v", err)
	}
}
