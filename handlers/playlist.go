package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

func Playlist(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/playlist")
	fullPath := fmt.Sprintf("%s%s", baseDir, path)
	log.Debug().Msgf("Requested: %s", fullPath)
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "Playlist not found", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, fullPath)
}
