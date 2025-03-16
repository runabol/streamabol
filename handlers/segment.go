package handlers

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/runabol/streamabol/stream"
)

func Segment(w http.ResponseWriter, r *http.Request) {
	re := regexp.MustCompile(`/segment/([0-9a-f]{32})/v0/(\d+)\.ts`)
	matches := re.FindStringSubmatch(r.URL.Path)
	if len(matches) < 3 {
		http.Error(w, "Invalid segment path", http.StatusBadRequest)
		return
	}
	playlistID := matches[1]
	segNum, err := strconv.Atoi(matches[2])
	if err != nil {
		http.Error(w, "Invalid segment number", http.StatusBadRequest)
		return
	}
	seg, err := stream.GetSegment(playlistID, segNum)
	if err != nil {
		http.Error(w, "Failed to get segment", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "video/mp2t")
	http.ServeFile(w, r, seg)
}
