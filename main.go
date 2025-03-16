package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/runabol/streamabol/handlers"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest", handlers.Manifest)
	mux.HandleFunc("/streams/", serveHLSFiles)
	handler := handlers.CORSMiddleware(mux)
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func serveHLSFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	fullPath := "/tmp" + path // Base path matches /tmp/streams/

	log.Printf("Requested: %s", fullPath)

	if strings.HasSuffix(path, ".m3u8") {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			http.Error(w, "Playlist not found", http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, fullPath)
		return
	}

	if strings.HasSuffix(path, ".ts") {
		w.Header().Set("Content-Type", "video/mp2t")
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			re := regexp.MustCompile(`/streams/([0-9a-f]{32})/v0/segment(\d+)\.ts`)
			matches := re.FindStringSubmatch(path)
			if len(matches) < 3 {
				http.Error(w, "Invalid segment path", http.StatusBadRequest)
				return
			}
			hash := matches[1] // MD5 hash from URL
			segNum, _ := strconv.Atoi(matches[2])
			startTime := segNum * 10
			duration := 10

			// Read the source URL from source.txt
			sourceFile := "/tmp/streams/" + hash + "/source.txt"
			src, err := os.ReadFile(sourceFile)
			if err != nil {
				http.Error(w, "Source URL not found", http.StatusInternalServerError)
				return
			}

			segDir := filepath.Dir(fullPath)
			os.MkdirAll(segDir, 0755)

			log.Printf("Encoding chunk: %s (start: %d, duration: %d)", fullPath, startTime, duration)
			err = encodeChunk(string(src), fullPath, startTime, duration)
			if err != nil {
				http.Error(w, "Failed to encode chunk", http.StatusInternalServerError)
				return
			}
		}
		http.ServeFile(w, r, fullPath)
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

func encodeChunk(src, outputPath string, startTime int, duration int) error {
	log.Printf("Encoding %s: start=%d, duration=%d", outputPath, startTime, duration)
	cmd := ffmpeg_go.Input(src, ffmpeg_go.KwArgs{
		"ss": startTime, // Seek to start time
	}).
		Output(outputPath, ffmpeg_go.KwArgs{
			"t":                duration,
			"c:v":              "libx264",
			"preset":           "ultrafast",
			"c:a":              "aac",
			"b:a":              "128k",
			"f":                "mpegts",
			"vf":               "scale=-2:720",
			"output_ts_offset": startTime,
		}).
		OverWriteOutput().
		WithErrorOutput(os.Stderr)

	log.Printf("FFmpeg command: %s", cmd.String())
	err := cmd.Run()
	if err != nil {
		log.Printf("Encode chunk error: %v", err)
	} else {
		log.Printf("Successfully encoded %s", outputPath)
	}
	return err
}
