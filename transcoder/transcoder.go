package transcoder

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Quality represents different video quality levels
type Quality struct {
	Name         string
	Width        int
	Height       int
	Bitrate      string
	AudioBitrate string
}

// Predefined quality levels
var Qualities = map[string]Quality{
	"1080p": {
		Name:         "1080p",
		Width:        1920,
		Height:       1080,
		Bitrate:      "4000k",
		AudioBitrate: "192k",
	},
	"720p": {
		Name:         "720p",
		Width:        1280,
		Height:       720,
		Bitrate:      "2500k",
		AudioBitrate: "128k",
	},
	"480p": {
		Name:         "480p",
		Width:        854,
		Height:       480,
		Bitrate:      "1000k",
		AudioBitrate: "96k",
	},
	"360p": {
		Name:         "360p",
		Width:        640,
		Height:       360,
		Bitrate:      "800k",
		AudioBitrate: "96k",
	},
}

// Transcoder handles video transcoding operations
type Transcoder struct {
	InputPath string
	OutputDir string
	Qualities []string
}

// NewTranscoder creates a new Transcoder instance
func NewTranscoder(inputPath string, outputDir string, qualities []string) *Transcoder {
	return &Transcoder{
		InputPath: inputPath,
		OutputDir: outputDir,
		Qualities: qualities,
	}
}

// Transcode converts the input video to different quality levels
func (t *Transcoder) Transcode() error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(t.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Get the base filename without extension
	baseName := strings.TrimSuffix(filepath.Base(t.InputPath), filepath.Ext(t.InputPath))

	// Transcode for each quality level
	for _, quality := range t.Qualities {
		q, exists := Qualities[quality]
		if !exists {
			log.Printf("Warning: Quality level %s not found, skipping", quality)
			continue
		}

		outputPath := filepath.Join(t.OutputDir, fmt.Sprintf("%s_%s.mp4", baseName, q.Name))

		// FFmpeg command for transcoding
		args := []string{
			"-i", t.InputPath,
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "23",
			"-vf", fmt.Sprintf("scale=%d:%d", q.Width, q.Height),
			"-b:v", q.Bitrate,
			"-c:a", "aac",
			"-b:a", q.AudioBitrate,
			"-movflags", "+faststart",
			outputPath,
		}

		cmd := exec.Command("ffmpeg", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		log.Printf("Transcoding to %s quality...", q.Name)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to transcode to %s: %v", q.Name, err)
		}
		log.Printf("Successfully transcoded to %s quality", q.Name)
	}

	return nil
}

// GetAvailableQualities returns a list of available quality levels for a video
func (t *Transcoder) GetAvailableQualities() []string {
	baseName := strings.TrimSuffix(filepath.Base(t.InputPath), filepath.Ext(t.InputPath))
	var available []string

	for _, quality := range t.Qualities {
		outputPath := filepath.Join(t.OutputDir, fmt.Sprintf("%s_%s.mp4", baseName, quality))
		if _, err := os.Stat(outputPath); err == nil {
			available = append(available, quality)
		}
	}

	return available
}
