package player

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"ytplayer/internal/downloader"
)

// QualityOption represents a selectable playback resolution preset.
type QualityOption struct {
	ID          string // "best", "1080", "720", "480", "360", "audio"
	Label       string // "Best / 4K", "1080p FHD", etc.
	Description string // "Max resolution (4K/1080p)", etc.
}

// QualityPresets contains all interactive quality options available to the user.
var QualityPresets = []QualityOption{
	{ID: "best", Label: "Best / 4K", Description: "Max resolution (4K / 1080p)"},
	{ID: "1080", Label: "1080p FHD", Description: "1080p Full High Definition"},
	{ID: "720", Label: "720p HD", Description: "720p High Definition"},
	{ID: "480", Label: "480p SD", Description: "480p Standard Definition"},
	{ID: "360", Label: "360p Saver", Description: "360p Data Saver"},
	{ID: "audio", Label: "Audio Only 🎵", Description: "Music / Podcasts (No Video)"},
}

// Player configures external video playback via mpv.
type Player struct {
	VideoOutput string // Custom video output driver (e.g. "tct", "sixel", "kitty", "gpu")
	Quality     string // Max video quality (e.g. "best", "1080", "720", "480", "360", "audio")
}

// NewPlayer initializes and returns a new Player instance.
func NewPlayer() *Player {
	return &Player{
		Quality: "best",
	}
}

// BuildMpvCmd constructs the exec.Cmd to play a video file or stream URL.
func (p *Player) BuildMpvCmd(target string, cookiesFile, cookiesFromBrowser string) *exec.Cmd {
	var args []string

	// Determine video output driver
	if p.VideoOutput != "" {
		args = append(args, "--vo="+p.VideoOutput)
	} else {
		// Auto-detect headless environment ONLY on Linux/BSD
		// macOS (darwin) and Windows have native graphical shells and do not use DISPLAY/WAYLAND_DISPLAY
		if runtime.GOOS == "linux" || runtime.GOOS == "freebsd" || runtime.GOOS == "openbsd" || runtime.GOOS == "netbsd" {
			display := os.Getenv("DISPLAY")
			waylandDisplay := os.Getenv("WAYLAND_DISPLAY")
			if display == "" && waylandDisplay == "" {
				// Headless: default to true-color terminal (tct) output
				args = append(args, "--vo=tct")
			}
		}
	}

	// If playing a remote youtube URL, pass cookies down to yt-dlp through mpv
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		// Set ytdl format selector based on quality setting
		switch p.Quality {
		case "audio":
			args = append(args, "--ytdl-format=bestaudio/best", "--no-video")
		case "1080":
			args = append(args, "--ytdl-format=bestvideo[height<=1080]+bestaudio/best[height<=1080]/best")
		case "720":
			args = append(args, "--ytdl-format=bestvideo[height<=720]+bestaudio/best[height<=720]/best")
		case "480":
			args = append(args, "--ytdl-format=bestvideo[height<=480]+bestaudio/best[height<=480]/best")
		case "360":
			args = append(args, "--ytdl-format=bestvideo[height<=360]+bestaudio/best[height<=360]/best/18")
		default:
			if p.Quality != "" && p.Quality != "best" && p.Quality != "max" {
				args = append(args, fmt.Sprintf("--ytdl-format=bestvideo[height<=%s]+bestaudio/best[height<=%s]/best", p.Quality, p.Quality))
			} else {
				args = append(args, "--ytdl-format=bestvideo+bestaudio/best")
			}
		}

		// Explicitly tell mpv where to find the yt-dlp executable we resolved (local bin or system path)
		if ytDlpPath, err := downloader.ResolvePath(); err == nil {
			if absPath, err := filepath.Abs(ytDlpPath); err == nil {
				args = append(args, "--script-opts=ytdl_hook-ytdl_path="+absPath)
			}
		}

		var ytdlOpts []string
		// Use visionos client to unlock Full HD / 4K streams and high quality Opus audio without PO token blocks
		ytdlOpts = append(ytdlOpts, "extractor-args=youtube:player_client=visionos")

		// Add JS runtime if node or deno is available on the system
		if _, err := exec.LookPath("node"); err == nil {
			ytdlOpts = append(ytdlOpts, "js-runtimes=node")
		} else if _, err := exec.LookPath("deno"); err == nil {
			ytdlOpts = append(ytdlOpts, "js-runtimes=deno")
		}

		if cookiesFile != "" {
			ytdlOpts = append(ytdlOpts, "cookies="+cookiesFile)
		}
		if cookiesFromBrowser != "" {
			ytdlOpts = append(ytdlOpts, "cookies-from-browser="+cookiesFromBrowser)
		}
		if len(ytdlOpts) > 0 {
			args = append(args, "--ytdl-raw-options="+strings.Join(ytdlOpts, ","))
		}
	}

	// Add the video file path or URL
	args = append(args, target)

	return exec.Command("mpv", args...)
}
