package downloader

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Video represents metadata of a YouTube search result.
type Video struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Uploader    string  `json:"uploader"`
	Duration    float64 `json:"duration"`
	URL         string  `json:"url"`
	Description string  `json:"description"`
	ViewCount   int64   `json:"view_count"`
}

// FormatDuration returns a string representation of the video duration (MM:SS or HH:MM:SS).
func (v Video) FormatDuration() string {
	d := int(v.Duration)
	if d <= 0 {
		return "0:00"
	}
	hours := d / 3600
	minutes := (d % 3600) / 60
	seconds := d % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// ResolvePath checks for yt-dlp in the local bin/ folder, and falls back to system PATH.
func ResolvePath() (string, error) {
	// 1. Check local bin/yt-dlp
	localPath := filepath.Join(".", "bin", "yt-dlp")
	if info, err := os.Stat(localPath); err == nil && !info.IsDir() {
		// Make sure it is executable
		if info.Mode()&0111 != 0 {
			return localPath, nil
		}
	}

	// 2. Check system PATH
	path, err := exec.LookPath("yt-dlp")
	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("yt-dlp not found in ./bin or system PATH")
}

// Search searches YouTube using yt-dlp.
func Search(ctx context.Context, query string, limit int, cookiesFile, cookiesFromBrowser string) ([]Video, error) {
	ytDlpPath, err := ResolvePath()
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 10
	}

	// Format query for ytsearch
	searchArg := fmt.Sprintf("ytsearch%d:%s", limit, query)

	args := []string{
		searchArg,
		"--dump-json",
		"--flat-playlist",
		"--no-warnings",
	}

	if cookiesFile != "" {
		args = append(args, "--cookies", cookiesFile)
	}
	if cookiesFromBrowser != "" {
		args = append(args, "--cookies-from-browser", cookiesFromBrowser)
	}

	cmd := exec.CommandContext(ctx, ytDlpPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	var videos []Video
	scanner := bufio.NewScanner(stdout)

	// Buffer can be large for detailed JSON, so let's allow up to 1MB lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var video Video
		if err := json.Unmarshal([]byte(line), &video); err != nil {
			// Skip malformed lines or other logs printed to stdout
			continue
		}

		// Ensure URL is populated
		if video.URL == "" && video.ID != "" {
			video.URL = "https://www.youtube.com/watch?v=" + video.ID
		}

		videos = append(videos, video)
	}

	if err := scanner.Err(); err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("error reading yt-dlp output: %w", err)
	}

	// Wait for process to exit
	if err := cmd.Wait(); err != nil {
		// ytsearch returns 1 if no results found, but let's check if we got videos anyway
		if len(videos) == 0 {
			return nil, fmt.Errorf("yt-dlp exited with error: %w", err)
		}
	}

	return videos, nil
}

// Download downloads a YouTube video as an MP3 audio file.
// progressChan receives progress percentage (0.0 to 100.0).
// Returns the absolute path to the downloaded MP3 file.
func Download(ctx context.Context, id string, progressChan chan<- float64, cookiesFile, cookiesFromBrowser string) (string, error) {
	ytDlpPath, err := ResolvePath()
	if err != nil {
		return "", err
	}

	// Ensure downloads directory exists
	downloadsDir := filepath.Join(".", "downloads")
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create downloads directory: %w", err)
	}

	outputPath := filepath.Join(downloadsDir, fmt.Sprintf("%s.mp3", id))

	// If the file already exists, we can return it directly.
	if info, err := os.Stat(outputPath); err == nil && !info.IsDir() {
		if progressChan != nil {
			progressChan <- 100.0
		}
		return filepath.Abs(outputPath)
	}

	videoURL := "https://www.youtube.com/watch?v=" + id

	// Command arguments for extracting audio
	args := []string{
		"-x",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"--no-keep-video",
		"--newline",
		"-o", filepath.Join(downloadsDir, "%(id)s.%(ext)s"),
	}

	if cookiesFile != "" {
		args = append(args, "--cookies", cookiesFile)
	}
	if cookiesFromBrowser != "" {
		args = append(args, "--cookies-from-browser", cookiesFromBrowser)
	}

	args = append(args, videoURL)

	cmd := exec.CommandContext(ctx, ytDlpPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	// Regexes to extract progress
	// Sample line: [download]  10.5% of 3.21MiB at 1.12MiB/s ETA 00:02
	progressRegex := regexp.MustCompile(`\[download\]\s+([0-9.]+)%`)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		if progressChan != nil {
			if strings.Contains(line, "[ExtractAudio]") {
				// Transitioning to extraction phase
				progressChan <- 99.0
			} else if matches := progressRegex.FindStringSubmatch(line); len(matches) > 1 {
				percent, err := strconv.ParseFloat(matches[1], 64)
				if err == nil {
					// Cap downloading progress at 95% to leave room for post-processing/conversion to MP3
					scaledPercent := percent * 0.95
					progressChan <- scaledPercent
				}
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}

	// Double check if file was created
	if _, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("download completed but file does not exist: %w", err)
	}

	if progressChan != nil {
		progressChan <- 100.0
	}

	return filepath.Abs(outputPath)
}
