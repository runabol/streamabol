package stream

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	ffmpego "github.com/u2takey/ffmpeg-go"
)

var baseDir = path.Join(os.TempDir(), "streamabol")

type streamInfo struct {
	Duration string `json:"duration"`
	Codec    string `json:"codec_name"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type formatInfo struct {
	Duration string `json:"duration"`
}

type probeResult struct {
	Streams []streamInfo `json:"streams"`
	Format  formatInfo   `json:"format"`
}

func GetManifest(src string) (string, error) {
	checksum := md5.Sum([]byte(src))
	hash := hex.EncodeToString(checksum[:]) // 32-char hex string
	outputDir := fmt.Sprintf("%s/%s", baseDir, hash)
	sourceFile := path.Join(outputDir, "source.txt")

	// Check if we've already processed this src
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		// Store the URL in the output directory
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			return "", errors.Wrapf(err, "Failed to create directory")
		}
		// Write the source URL to a file
		if err := os.WriteFile(sourceFile, []byte(src), 0644); err != nil {
			return "", errors.Wrapf(err, "Failed to write source file")
		}
		// Generate playlists
		if err := generatePlaylist(src, outputDir, hash); err != nil {
			return "", errors.Wrapf(err, "Failed to generate playlist")
		}
	}

	return fmt.Sprintf("%s/%s/master.m3u8", baseDir, hash), nil
}

func GetPlaylist(id string) (string, error) {
	fullPath := fmt.Sprintf("%s/%s/v0.m3u8", baseDir, id)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", errors.Errorf("Playlist not found")
	}
	return fullPath, nil
}

func generatePlaylist(src, outputDir, hash string) error {
	duration, err := getDuration(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir+"/v0", 0755); err != nil {
		return errors.Wrapf(err, "Failed to create directory: %s", outputDir+"/v0")
	}

	// Write master.m3u8 with relative path
	masterContent := fmt.Sprintf(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-STREAM-INF:BANDWIDTH=561911,AVERAGE-BANDWIDTH=497690,RESOLUTION=640x360,CODECS="avc1.64001e,mp4a.40.2"
/playlist/%s/v0.m3u8
`, hash)
	if err := os.WriteFile(outputDir+"/master.m3u8", []byte(masterContent), 0644); err != nil {
		log.Printf("Error writing master.m3u8: %v", err)
		return err
	}

	// Generate v0.m3u8 with segment entries
	segmentDuration := 4.0
	numSegments := int(duration / segmentDuration)
	if duration-float64(numSegments)*segmentDuration > 0 {
		numSegments++
	}

	var v0Content strings.Builder
	v0Content.WriteString("#EXTM3U\n")
	v0Content.WriteString("#EXT-X-VERSION:3\n")
	v0Content.WriteString("#EXT-X-TARGETDURATION:4\n")
	v0Content.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
	v0Content.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")

	for i := 0; i < numSegments; i++ {
		remaining := duration - float64(i)*segmentDuration
		segDur := segmentDuration
		if remaining < segmentDuration {
			segDur = remaining
		}
		if _, err := v0Content.WriteString("#EXTINF:" + strconv.FormatFloat(segDur, 'f', 3, 64) + ",\n"); err != nil {
			return errors.Wrapf(err, "Failed to write segment duration: %v", err)
		}
		if _, err := v0Content.WriteString(fmt.Sprintf("/segment/%s/v0/%d.ts\n", hash, i)); err != nil {
			return errors.Wrapf(err, "Failed to write segment entry: %v", err)
		}
	}
	v0Content.WriteString("#EXT-X-ENDLIST\n")

	if err := os.WriteFile(outputDir+"/v0.m3u8", []byte(v0Content.String()), 0644); err != nil {
		return errors.Wrapf(err, "Failed to write manifest file: %v", err)
	}

	log.Debug().Msgf("Generated playlists in %s for src=%s", outputDir, src)
	return nil
}

func GetSegment(playlistID string, segNum int) (string, error) {
	fullPath := fmt.Sprintf("%s/%s/v0/%d.ts", baseDir, playlistID, segNum)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		startTime := segNum * 4
		duration := 4

		// Read the source URL from source.txt
		sourceFile := fmt.Sprintf("%s/%s/source.txt", baseDir, playlistID)
		src, err := os.ReadFile(sourceFile)
		if err != nil {
			return "", errors.Wrapf(err, "Source URL not found")
		}

		segDir := filepath.Dir(fullPath)
		if err := os.MkdirAll(segDir, 0755); err != nil {
			return "", errors.Wrapf(err, "Failed to create directory: %s", segDir)
		}

		log.Debug().Msgf("Encoding chunk: %s (start: %d, duration: %d)", fullPath, startTime, duration)
		err = encodeChunk(string(src), fullPath, startTime, duration)
		if err != nil {
			return "", errors.Wrapf(err, "Failed to encode chunk")
		}
	}
	return fullPath, nil
}

func encodeChunk(src, outputPath string, startTime int, duration int) error {
	log.Debug().Msgf("Encoding %s: start=%d, duration=%d", outputPath, startTime, duration)
	cmd := ffmpego.Input(src, ffmpego.KwArgs{
		"ss": startTime, // Seek to start time
	}).
		Output(outputPath, ffmpego.KwArgs{
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

	log.Debug().Msgf("FFmpeg command: %s", cmd.String())
	err := cmd.Run()
	if err != nil {
		log.Error().Err(err).Msgf("Encode chunk error: %v", err)
		return errors.Wrapf(err, "Failed to encode chunk: %s", outputPath)
	}
	return nil
}

func getDuration(src string) (float64, error) {
	output, err := ffmpego.Probe(src)
	if err != nil {
		log.Error().Err(err).Msgf("Probe error: %v", err)
		return 0, err
	}

	var result probeResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		log.Error().Err(err).Msgf("Unmarshal error: %v", err)
		return 0, err
	}

	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		log.Error().Err(err).Msgf("Duration parse error: %v", err)
		return 0, err
	}
	return duration, nil
}
