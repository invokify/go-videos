# Go Video Gallery

A modern video gallery application built with Go, featuring a responsive grid layout and interactive video playback.

## Features

- Responsive video grid layout that adapts to different screen sizes
- Hover-to-play video previews
- Interactive play/pause controls
- Video streaming with support for various video formats
- Automatic video metadata extraction (duration, title)
- Automatic thumbnail generation for video previews
- Modern and clean user interface
- HTTP caching support for improved video playback performance
- Range request support for efficient video streaming

## Prerequisites

- Go 1.16 or higher
- FFmpeg (for video metadata extraction)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/go-video.git
cd go-video
```

2. Install dependencies:
```bash
go mod download
```

3. Create a `videos` directory in the project root:
```bash
mkdir videos
```

## Usage

1. Place your video files in the `videos` directory. Supported formats include:
   - MP4
   - WebM
   - MKV
   - QuickTime
   - AVI

2. Start the server:
```bash
go run main.go
```

3. Open your browser and navigate to:
```
http://localhost:8080/videos
```

## API Endpoints

- `GET /videos` - Displays the video gallery interface
- `GET /stream/{filename}` - Streams a video file with support for:
  - HTTP caching (1-hour cache duration)
  - Range requests for partial content
  - ETag and Last-Modified validation

## Project Structure

```
go-video/
├── main.go           # Application entry point
├── handlers/         # HTTP request handlers
│   ├── list_videos.go
│   └── video_handler.go
├── templates/        # HTML templates
│   ├── base.html
│   └── videos.html
├── videos/          # Video storage directory
├── thumbnails/      # Generated video thumbnails
├── go.mod           # Go module definition
└── go.sum           # Go module checksums
```

## Development

### Adding New Features

1. Create new handlers in the `handlers` directory
2. Add corresponding templates in the `templates` directory
3. Update routes in `main.go`

### Styling

The application uses modern CSS features:
- CSS Grid for responsive layout
- Flexbox for alignment
- CSS transitions for smooth animations
- Modern CSS properties for visual effects

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [Go](https://golang.org/) - The programming language
- [FFmpeg](https://ffmpeg.org/) - For video processing
- [h2non/filetype](https://github.com/h2non/filetype) - For file type detection 