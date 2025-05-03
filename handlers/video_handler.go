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
	"time"

	"go-video/transcoder"

	"html/template"

	"golang.org/x/time/rate"
)

// Global rate limiter for video streaming
var streamLimiter = rate.NewLimiter(rate.Limit(10), 5) // 10 requests per second, burst of 5

// Buffer size for video streaming (1MB)
const streamBufferSize = 1 << 20

func StreamHandler(w http.ResponseWriter, r *http.Request) {
	// Apply rate limiting
	if !streamLimiter.Allow() {
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		return
	}

	// Extract the video filename and quality from the URL
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/stream/"), "/")
	if len(pathParts) < 1 {
		http.Error(w, "File not specified", http.StatusBadRequest)
		return
	}

	filename := pathParts[0]
	quality := "original"
	if len(pathParts) > 1 {
		quality = pathParts[1]
	}

	// Construct the path to the video file
	var path string
	if quality == "original" {
		path = filepath.Join("videos", filename)
	} else {
		// For transcoded versions, the filename format is: original_name_quality.mp4
		baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
		path = filepath.Join("videos", fmt.Sprintf("%s_%s.mp4", baseName, quality))
	}

	// Open the video file
	video, err := os.Open(path)
	if err != nil {
		// If the requested quality is not available, try the original
		if quality != "original" {
			path = filepath.Join("videos", filename)
			video, err = os.Open(path)
			if err != nil {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}
		} else {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
	}
	defer video.Close()

	// Get video file information
	fileInfo, err := video.Stat()
	if err != nil {
		http.Error(w, "Failed to get file info", http.StatusInternalServerError)
		return
	}
	fileSize := fileInfo.Size()
	lastModified := fileInfo.ModTime()

	// Determine content type based on file extension
	contentType := "video/mp4" // default
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".webm":
		contentType = "video/webm"
	case ".ogg":
		contentType = "video/ogg"
	case ".mov":
		contentType = "video/quicktime"
	}

	// Set headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.Header().Set("ETag", fmt.Sprintf(`"%x-%x"`, lastModified.Unix(), fileSize))
	w.Header().Set("Last-Modified", lastModified.UTC().Format(http.TimeFormat))
	w.Header().Set("Accept-Ranges", "bytes")

	// Check if the client's cached version is still valid
	if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
		if ifNoneMatch == fmt.Sprintf(`"%x-%x"`, lastModified.Unix(), fileSize) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	if ifModifiedSince := r.Header.Get("If-Modified-Since"); ifModifiedSince != "" {
		if t, err := time.Parse(http.TimeFormat, ifModifiedSince); err == nil {
			if lastModified.Before(t.Add(1 * time.Second)) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
	}

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

		// Validate range
		if start < 0 || end >= fileSize || start > end {
			http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
			return
		}

		// Seek to the start position
		if _, err := video.Seek(start, 0); err != nil {
			http.Error(w, "Failed to seek file", http.StatusInternalServerError)
			return
		}

		// Set headers for partial content
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)

		// Stream the video chunk with custom buffer
		buffer := make([]byte, streamBufferSize)
		remaining := end - start + 1
		for remaining > 0 {
			readSize := streamBufferSize
			if remaining < int64(readSize) {
				readSize = int(remaining)
			}
			n, err := video.Read(buffer[:readSize])
			if err != nil && err != io.EOF {
				log.Printf("Error streaming video: %v", err)
				return
			}
			if n > 0 {
				if _, err := w.Write(buffer[:n]); err != nil {
					log.Printf("Error writing video chunk: %v", err)
					return
				}
				remaining -= int64(n)
			}
			if err == io.EOF {
				break
			}
		}
	} else {
		// Stream entire file if no range is specified
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))

		// Stream with custom buffer
		buffer := make([]byte, streamBufferSize)
		remaining := fileSize
		for remaining > 0 {
			readSize := streamBufferSize
			if remaining < int64(readSize) {
				readSize = int(remaining)
			}
			n, err := video.Read(buffer[:readSize])
			if err != nil && err != io.EOF {
				log.Printf("Error streaming video: %v", err)
				return
			}
			if n > 0 {
				if _, err := w.Write(buffer[:n]); err != nil {
					log.Printf("Error writing video chunk: %v", err)
					return
				}
				remaining -= int64(n)
			}
			if err == io.EOF {
				break
			}
		}
	}
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the multipart form with a 10MB max memory limit
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get the file from the form
	file, header, err := r.FormFile("video")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check if the file is a video
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "video/") {
		http.Error(w, "File must be a video", http.StatusBadRequest)
		return
	}

	// Create videos directory if it doesn't exist
	if err := os.MkdirAll("videos", 0755); err != nil {
		http.Error(w, "Failed to create videos directory", http.StatusInternalServerError)
		return
	}

	// Create the file in the videos directory
	filename := header.Filename
	filepath := filepath.Join("videos", filename)
	dst, err := os.Create(filepath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the destination file
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Create transcoded versions
	trans := transcoder.NewTranscoder(
		filepath,
		"videos",
		[]string{"1080p", "720p", "480p", "360p"},
	)

	// Start transcoding in a goroutine
	go func() {
		if err := trans.Transcode(); err != nil {
			log.Printf("Error transcoding video: %v", err)
		}
	}()

	// Redirect back to the videos page
	http.Redirect(w, r, "/videos", http.StatusSeeOther)
}

func VideoPlayerHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the video filename from the URL
	filename := strings.TrimPrefix(r.URL.Path, "/player/")
	if filename == "" {
		http.Error(w, "File not specified", http.StatusBadRequest)
		return
	}

	// Create transcoder instance to get available qualities
	trans := transcoder.NewTranscoder(
		filepath.Join("videos", filename),
		"videos",
		[]string{"1080p", "720p", "480p", "360p"},
	)

	// Get available qualities
	availableQualities := trans.GetAvailableQualities()

	// Parse and execute the template
	tmpl, err := template.ParseFiles("templates/video.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	data := struct {
		Filename           string
		AvailableQualities []string
	}{
		Filename:           filename,
		AvailableQualities: availableQualities,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}
