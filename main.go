package main

import (
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/runabol/streamabol/handlers"
	"github.com/runabol/streamabol/logging"
)

func main() {
	if err := logging.Setup(); err != nil {
		log.Fatal().Err(err).Msg("Failed to setup logging")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest.m3u8", handlers.Manifest)
	mux.HandleFunc("/playlist/", handlers.Playlist)
	mux.HandleFunc("/segment/", handlers.Segment)
	handler := handlers.CORSMiddleware(handlers.LoggerMiddleware(mux))
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
