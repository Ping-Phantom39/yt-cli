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
	"runtime"
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
	// 1. Check local bin/yt-dlp (with .exe on Windows)
	ytDlpName := "yt-dlp"
	if runtime.GOOS == "windows" {
		ytDlpName = "yt-dlp.exe"
	}
	localPath := filepath.Join(".", "bin", ytDlpName)
	if info, err := os.Stat(localPath); err == nil && !info.IsDir() {
		// Make sure it is executable
		if runtime.GOOS == "windows" || info.Mode()&0111 != 0 {
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
		"--js-runtimes", "node",
		"--remote-components", "ejs:github",
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

// Download downloads a YouTube video as an MP4 file.
// progressChan receives progress percentage (0.0 to 100.0).
// Returns the absolute path to the downloaded MP4 file.
func Download(ctx context.Context, id string, outputPath string, progressChan chan<- float64, cookiesFile, cookiesFromBrowser string) (string, error) {
	ytDlpPath, err := ResolvePath()
	if err != nil {
		return "", err
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// If the file already exists and is not empty, we can return it directly.
	if info, err := os.Stat(outputPath); err == nil && !info.IsDir() && info.Size() > 0 {
		if progressChan != nil {
			progressChan <- 100.0
		}
		absPath, err := filepath.Abs(outputPath)
		if err != nil {
			return outputPath, nil
		}
		return absPath, nil
	}

	videoURL := "https://www.youtube.com/watch?v=" + id

	// Generate output template based on destination path
	ext := filepath.Ext(outputPath)
	var outputTemplate string
	if ext != "" {
		outputTemplate = outputPath[:len(outputPath)-len(ext)] + ".%(ext)s"
	} else {
		outputTemplate = outputPath + ".%(ext)s"
	}

	// Command arguments for downloading best video and audio merged into mp4
	args := []string{
		"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
		"--merge-output-format", "mp4",
		"--newline",
		"--js-runtimes", "node",
		"--remote-components", "ejs:github",
		"-o", outputTemplate,
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

	// Capture stderr so we can include the actual error message
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	// Regex to extract progress percentage
	progressRegex := regexp.MustCompile(`\[download\]\s+([0-9.]+)%`)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		if progressChan != nil {
			if matches := progressRegex.FindStringSubmatch(line); len(matches) > 1 {
				percent, err := strconv.ParseFloat(matches[1], 64)
				if err == nil {
					// Scale down downloading to 98% to leave room for post-processing / merge phase
					scaledPercent := percent * 0.98
					progressChan <- scaledPercent
				}
			} else if strings.Contains(line, "[Merger]") {
				progressChan <- 99.0
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		errDetail := strings.TrimSpace(stderrBuf.String())
		if errDetail != "" {
			for _, line := range strings.Split(errDetail, "\n") {
				if strings.Contains(line, "ERROR") {
					errDetail = strings.TrimSpace(line)
				}
			}
			return "", fmt.Errorf("%s", errDetail)
		}
		return "", fmt.Errorf("failed to download: %w", err)
	}

	// Verify the final output path exists
	if _, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("download completed but target file does not exist: %w", err)
	}

	if progressChan != nil {
		progressChan <- 100.0
	}

	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return outputPath, nil
	}
	return absPath, nil
}
