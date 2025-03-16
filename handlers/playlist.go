package handlers

import (
	"net/http"
	"strings"

	"github.com/runabol/streamabol/stream"
)

// Playlist handles requests for HLS playlist files
func Playlist(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/playlist"), "/v0.m3u8")
	playlist, error := stream.GetPlaylist(id)
	if error != nil {
		http.Error(w, "Playlist not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	http.ServeFile(w, r, playlist)
}
