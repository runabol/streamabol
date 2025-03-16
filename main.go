package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/runabol/streamabol/handlers"
)

func main() {
	if err := setupLogging(); err != nil {
		log.Fatal().Err(err).Msg("Failed to setup logging")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest", handlers.Manifest)
	mux.HandleFunc("/playlist/", handlers.Playlist)
	mux.HandleFunc("/segment/", handlers.Segment)
	handler := handlers.CORSMiddleware(handlers.LoggerMiddleware(mux))
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}

func setupLogging() error {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "debug"
	}
	logLevel := strings.ToLower(level)
	// setup log level
	switch logLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn", "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		return errors.Errorf("invalid logging level: %s", logLevel)
	}
	// setup log format (pretty / json)
	format := os.Getenv("LOG_FORMAT")
	if format == "" {
		format = "pretty"
	}
	logFormat := strings.ToLower(format)
	switch logFormat {
	case "pretty":
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05"})
	case "json":
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	default:
		return errors.Errorf("invalid logging format: %s", logFormat)
	}
	return nil
}
