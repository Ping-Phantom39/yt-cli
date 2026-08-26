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
		if strings.HasPrefix(v.ID, "local:") {
			return "LOCAL"
		}
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

// SanitizeFilename cleans a string to make it safe for use as a filename across operating systems.
func SanitizeFilename(name string) string {
	// Replace illegal filename characters (\, /, :, *, ?, ", <, >, |, NUL, control chars) with '_'
	invalidChars := regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)
	sanitized := invalidChars.ReplaceAllString(name, "_")

	// Collapse spaces and underscores surrounding invalid characters
	multiSpaceUnderscore := regexp.MustCompile(`[ _]*_[ _]*`)
	sanitized = multiSpaceUnderscore.ReplaceAllString(sanitized, "_")

	// Trim leading and trailing spaces, dots, and underscores
	sanitized = strings.Trim(sanitized, " ._")

	// Limit length to 200 runes
	runes := []rune(sanitized)
	if len(runes) > 200 {
		sanitized = strings.Trim(string(runes[:200]), " ._")
	}

	if sanitized == "" {
		return "video"
	}
	return sanitized
}

// ScanLocalVideos scans specified directories for local video files.
func ScanLocalVideos(dirs ...string) ([]Video, error) {
	var videos []Video
	seen := make(map[string]bool)

	validExts := map[string]bool{
		".mp4":  true,
		".mkv":  true,
		".avi":  true,
		".webm": true,
		".mov":  true,
		".flv":  true,
		".wmv":  true,
		".m4v":  true,
		".ts":   true,
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if !validExts[ext] {
				continue
			}

			fullPath := filepath.Join(dir, entry.Name())
			absPath, err := filepath.Abs(fullPath)
			if err != nil {
				absPath = fullPath
			}

			if seen[absPath] {
				continue
			}
			seen[absPath] = true

			// Title is file name without extension
			title := entry.Name()
			if ext != "" {
				title = title[:len(title)-len(ext)]
			}

			folderName := filepath.Base(dir)
			if dir == "." {
				folderName = "Current Dir"
			}

			video := Video{
				ID:          "local:" + absPath,
				Title:       title,
				Uploader:    fmt.Sprintf("Local (%s)", folderName),
				URL:         absPath,
				Description: fmt.Sprintf("Local video file: %s", fullPath),
			}

			videos = append(videos, video)
		}
	}

	return videos, nil
}

// ResolvePath checks for yt-dlp in local bin/, executable dir, and falls back to system PATH.
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

	// 2. Check next to running executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		exeAdjacent := filepath.Join(exeDir, ytDlpName)
		if info, err := os.Stat(exeAdjacent); err == nil && !info.IsDir() {
			if runtime.GOOS == "windows" || info.Mode()&0111 != 0 {
				return exeAdjacent, nil
			}
		}
		exeBin := filepath.Join(exeDir, "bin", ytDlpName)
		if info, err := os.Stat(exeBin); err == nil && !info.IsDir() {
			if runtime.GOOS == "windows" || info.Mode()&0111 != 0 {
				return exeBin, nil
			}
		}
	}

	// 3. Check system PATH
	path, err := exec.LookPath("yt-dlp")
	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("yt-dlp not found in ./bin or system PATH")
}

// ResolveFFmpegPath checks for ffmpeg in the local bin/ folder, executable dir, and falls back to system PATH.
func ResolveFFmpegPath() (string, error) {
	ffmpegName := "ffmpeg"
	if runtime.GOOS == "windows" {
		ffmpegName = "ffmpeg.exe"
	}
	localPath := filepath.Join(".", "bin", ffmpegName)
	if info, err := os.Stat(localPath); err == nil && !info.IsDir() {
		if runtime.GOOS == "windows" || info.Mode()&0111 != 0 {
			return localPath, nil
		}
	}

	// Check next to running executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		exeAdjacent := filepath.Join(exeDir, ffmpegName)
		if info, err := os.Stat(exeAdjacent); err == nil && !info.IsDir() {
			if runtime.GOOS == "windows" || info.Mode()&0111 != 0 {
				return exeAdjacent, nil
			}
		}
		exeBin := filepath.Join(exeDir, "bin", ffmpegName)
		if info, err := os.Stat(exeBin); err == nil && !info.IsDir() {
			if runtime.GOOS == "windows" || info.Mode()&0111 != 0 {
				return exeBin, nil
			}
		}
	}

	path, err := exec.LookPath("ffmpeg")
	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("ffmpeg not found in ./bin or system PATH")
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
		"--extractor-args", "youtube:player_client=visionos,android_creator,tv_embedded,android,ios,tv,web",
		"--remote-components", "ejs:github",
	}

	if _, err := exec.LookPath("node"); err == nil {
		args = append(args, "--js-runtimes", "node")
	} else if _, err := exec.LookPath("deno"); err == nil {
		args = append(args, "--js-runtimes", "deno")
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

// Download downloads a YouTube video as an MP4 file with the specified quality.
// progressChan receives progress percentage (0.0 to 100.0).
// Returns the absolute path to the downloaded MP4 file.
func Download(ctx context.Context, id string, outputPath string, quality string, progressChan chan<- float64, cookiesFile, cookiesFromBrowser string) (string, error) {
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

	// Select download format string based on requested quality
	var formatSelector string
	switch quality {
	case "audio":
		formatSelector = "bestaudio/best"
	case "1080":
		formatSelector = "bestvideo[height<=1080][ext=mp4]+bestaudio[ext=m4a]/bestvideo[height<=1080]+bestaudio/best[height<=1080]/best"
	case "720":
		formatSelector = "bestvideo[height<=720][ext=mp4]+bestaudio[ext=m4a]/bestvideo[height<=720]+bestaudio/best[height<=720]/best"
	case "480":
		formatSelector = "bestvideo[height<=480][ext=mp4]+bestaudio[ext=m4a]/bestvideo[height<=480]+bestaudio/best[height<=480]/best"
	case "360":
		formatSelector = "bestvideo[height<=360][ext=mp4]+bestaudio[ext=m4a]/bestvideo[height<=360]+bestaudio/best[height<=360]/best/18"
	default:
		if quality != "" && quality != "best" && quality != "max" {
			formatSelector = fmt.Sprintf("bestvideo[height<=%s][ext=mp4]+bestaudio[ext=m4a]/bestvideo[height<=%s]+bestaudio/best[height<=%s]/best", quality, quality, quality)
		} else {
			formatSelector = "bestvideo[ext=mp4]+bestaudio[ext=m4a]/bestvideo+bestaudio/best[ext=mp4]/best/18"
		}
	}

	// Command arguments for downloading video and audio merged into mp4
	args := []string{
		"--no-playlist",
		"--extractor-args", "youtube:player_client=visionos,android_creator,tv_embedded,android,ios,tv,web",
		"-f", formatSelector,
		"--merge-output-format", "mp4",
		"--newline",
		"--remote-components", "ejs:github",
	}

	if _, err := exec.LookPath("node"); err == nil {
		args = append(args, "--js-runtimes", "node")
	} else if _, err := exec.LookPath("deno"); err == nil {
		args = append(args, "--js-runtimes", "deno")
	}

	ffmpegPath, ffmpegErr := ResolveFFmpegPath()
	if ffmpegErr == nil {
		args = append(args, "--ffmpeg-location", ffmpegPath)
	}

	if cookiesFile != "" {
		args = append(args, "--cookies", cookiesFile)
	}
	if cookiesFromBrowser != "" {
		args = append(args, "--cookies-from-browser", cookiesFromBrowser)
	}

	args = append(args, "-o", outputTemplate, videoURL)

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

	ytErr := cmd.Wait()

	// Helper function to attempt fallback merging if yt-dlp left unmerged stream files
	tryFallbackMerge := func() bool {
		if ffmpegErr != nil {
			return false
		}

		baseName := outputPath
		if ext != "" {
			baseName = outputPath[:len(outputPath)-len(ext)]
		}
		basePrefix := filepath.Base(baseName)

		entries, err := os.ReadDir(destDir)
		if err != nil {
			return false
		}

		var videoFile, audioFile string
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasPrefix(name, basePrefix) {
				full := filepath.Join(destDir, name)
				lower := strings.ToLower(name)
				// Look for temporary format files such as Title.f399.mp4, Title.f140.m4a
				if strings.Contains(lower, ".f") || strings.Contains(lower, ".temp.") {
					if strings.HasSuffix(lower, ".m4a") || strings.HasSuffix(lower, ".aac") || strings.HasSuffix(lower, ".opus") || (strings.HasSuffix(lower, ".webm") && strings.Contains(lower, "audio")) {
						audioFile = full
					} else if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".webm") || strings.HasSuffix(lower, ".mkv") {
						videoFile = full
					}
				}
			}
		}

		if videoFile != "" && audioFile != "" {
			if progressChan != nil {
				progressChan <- 99.0
			}
			cmdMerge := exec.CommandContext(ctx, ffmpegPath, "-y", "-i", videoFile, "-i", audioFile, "-c:v", "copy", "-c:a", "aac", "-strict", "experimental", outputPath)
			if err := cmdMerge.Run(); err == nil {
				if info, err := os.Stat(outputPath); err == nil && info.Size() > 0 {
					_ = os.Remove(videoFile)
					_ = os.Remove(audioFile)
					return true
				}
			}
		}
		return false
	}

	// Verify target file exists, otherwise attempt fallback merge
	if _, err := os.Stat(outputPath); err != nil || func() bool { info, e := os.Stat(outputPath); return e == nil && info.Size() == 0 }() {
		if !tryFallbackMerge() {
			if ytErr != nil {
				errDetail := strings.TrimSpace(stderrBuf.String())
				if errDetail != "" {
					for _, line := range strings.Split(errDetail, "\n") {
						if strings.Contains(line, "ERROR") {
							errDetail = strings.TrimSpace(line)
						}
					}
					return "", fmt.Errorf("%s", errDetail)
				}
				return "", fmt.Errorf("failed to download: %w", ytErr)
			}
			return "", fmt.Errorf("download completed but target file does not exist: %w", os.ErrNotExist)
		}
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
