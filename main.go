package main

import (
	"log"
	"net/http"

	"github.com/runabol/streamabol/handlers"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest", handlers.Manifest)
	mux.HandleFunc("/playlist/", handlers.Playlist)
	handler := handlers.CORSMiddleware(mux)
	log.Fatal(http.ListenAndServe(":8080", handler))
}
