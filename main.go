package main

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/runabol/streamabol/logging"
	"github.com/runabol/streamabol/server"
)

func main() {
	if err := logging.Setup(); err != nil {
		log.Fatal().Err(err).Msg("Failed to setup logging")
	}
	address := os.Getenv("ADDRESS")
	if address == "" {
		address = "0.0.0.0:8080"
	}
	log.Info().Msgf("Starting server on %s", address)
	srv := server.NewServer(":8080")
	if err := srv.Start(); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
