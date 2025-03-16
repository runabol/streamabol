package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

func Segment(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/segment")
	fullPath := fmt.Sprintf("%s%s", baseDir, path)

	log.Printf("Requested: %s", fullPath)

	w.Header().Set("Content-Type", "video/mp2t")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		re := regexp.MustCompile(`/([0-9a-f]{32})/v0/(\d+)\.ts`)
		matches := re.FindStringSubmatch(path)
		if len(matches) < 3 {
			http.Error(w, "Invalid segment path", http.StatusBadRequest)
			return
		}
		hash := matches[1] // MD5 hash from URL
		segNum, _ := strconv.Atoi(matches[2])
		startTime := segNum * 4
		duration := 4

		// Read the source URL from source.txt
		sourceFile := baseDir + "/" + hash + "/source.txt"
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
		OverWriteOutput()

	log.Printf("FFmpeg command: %s", cmd.String())
	err := cmd.Run()
	if err != nil {
		log.Printf("Encode chunk error: %v", err)
	} else {
		log.Printf("Successfully encoded %s", outputPath)
	}
	return err
}
