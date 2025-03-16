package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func Playlist(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/playlist")
	fullPath := fmt.Sprintf("%s%s", baseDir, path)
	log.Printf("Requested: %s", fullPath)
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "Playlist not found", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, fullPath)
}
