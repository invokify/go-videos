package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func StreamHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the video filename from the URL
	filename := strings.TrimPrefix(r.URL.Path, "/stream/")
	if filename == "" {
		http.Error(w, "File not specified", http.StatusBadRequest)
		return
	}

	// Construct the path to the video file
	path := filepath.Join("videos", filename)

	// Open the video file
	video, err := os.Open(path)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer video.Close()

	// Get video file information
	fileInfo, err := video.Stat()
	if err != nil {
		http.Error(w, "Failed to get file info", http.StatusInternalServerError)
		return
	}
	fileSize := fileInfo.Size()

	// Get the range header
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		// Parse the range header
		ranges := strings.Split(strings.TrimPrefix(rangeHeader, "bytes="), "-")
		if len(ranges) != 2 {
			http.Error(w, "Invalid range header", http.StatusBadRequest)
			return
		}

		// Parse start and end positions
		start, err := strconv.ParseInt(ranges[0], 10, 64)
		if err != nil {
			http.Error(w, "Invalid range header", http.StatusBadRequest)
			return
		}

		var end int64
		if ranges[1] == "" {
			end = fileSize - 1
		} else {
			end, err = strconv.ParseInt(ranges[1], 10, 64)
			if err != nil {
				http.Error(w, "Invalid range header", http.StatusBadRequest)
				return
			}
		}

		// Seek to the start position
		if _, err := video.Seek(start, 0); err != nil {
			http.Error(w, "Failed to seek file", http.StatusInternalServerError)
			return
		}

		// Set headers for partial content
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)

		// Stream the video chunk
		_, err = io.CopyN(w, video, end-start+1)
		if err != nil {
			log.Printf("Error streaming video: %v", err)
			return
		}
	} else {
		// Stream entire file if no range is specified
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
		w.Header().Set("Accept-Ranges", "bytes")
		_, err = io.Copy(w, video)
		if err != nil {
			log.Printf("Error streaming video: %v", err)
			return
		}
	}
}
