package main

import (
	"net/http"
	"os"

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
	address := os.Getenv("ADDRESS")
	if address == "" {
		address = "0.0.0.0:8080"
	}
	log.Info().Msgf("Starting server on %s", address)
	if err := http.ListenAndServe(address, handler); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
