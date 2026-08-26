package player

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"ytplayer/internal/downloader"
)

// Player configures external video playback via mpv.
type Player struct {
	VideoOutput string // Custom video output driver (e.g. "tct", "sixel", "kitty", "gpu")
}

// NewPlayer initializes and returns a new Player instance.
func NewPlayer() *Player {
	return &Player{}
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
		// Explicitly tell mpv where to find the yt-dlp executable we resolved (local bin or system path)
		if ytDlpPath, err := downloader.ResolvePath(); err == nil {
			if absPath, err := filepath.Abs(ytDlpPath); err == nil {
				args = append(args, "--script-opts=ytdl_hook-ytdl_path="+absPath)
			}
		}

		var ytdlOpts []string
		// Use android player client to bypass 403 Forbidden / 429 Too Many Requests / SABR format errors on YouTube
		ytdlOpts = append(ytdlOpts, "extractor-args=youtube:player_client=android")

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
