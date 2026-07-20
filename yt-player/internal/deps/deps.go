package deps

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Dependency represents a system or binary dependency
type Dependency struct {
	Name                string
	Found               bool
	Version             string
	Path                string
	Required            bool
	Description         string
	InstallInstructions string
}

// CheckAll verifies all runtime dependencies of ytplayer
func CheckAll() []Dependency {
	var results []Dependency

	// 1. Check yt-dlp
	ytDlpDep := Dependency{
		Name:        "yt-dlp",
		Required:    true,
		Description: "YouTube media downloader and metadata scraper",
		InstallInstructions: "Download from https://github.com/yt-dlp/yt-dlp or place in ./bin/yt-dlp",
	}
	ytDlpPath, err := resolveYtDlp()
	if err == nil {
		ytDlpDep.Found = true
		ytDlpDep.Path = ytDlpPath
		// Get version
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, ytDlpPath, "--version")
		if out, err := cmd.Output(); err == nil {
			ytDlpDep.Version = strings.TrimSpace(string(out))
		} else {
			ytDlpDep.Version = "Unknown (execution failed)"
		}
	}
	results = append(results, ytDlpDep)

	// 2. Check ffmpeg
	ffmpegDep := Dependency{
		Name:        "ffmpeg",
		Required:    true,
		Description: "Media transcoder (needed by yt-dlp to merge video and audio streams)",
		InstallInstructions: "sudo apt install -y ffmpeg (Debian/Ubuntu) or brew install ffmpeg (macOS)",
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		ffmpegDep.Found = true
		ffmpegDep.Path = p
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, p, "-version")
		if out, err := cmd.Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 0 {
				ffmpegDep.Version = strings.TrimSpace(lines[0])
			}
		}
	}
	results = append(results, ffmpegDep)

	// 3. Check mpv
	mpvDep := Dependency{
		Name:        "mpv",
		Required:    true,
		Description: "Native terminal/GUI media player used to render and stream video",
		InstallInstructions: "sudo apt install -y mpv (Debian/Ubuntu) or brew install mpv (macOS)",
	}
	if p, err := exec.LookPath("mpv"); err == nil {
		mpvDep.Found = true
		mpvDep.Path = p
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, p, "--version")
		if out, err := cmd.Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 0 {
				mpvDep.Version = strings.TrimSpace(lines[0])
			}
		}
	}
	results = append(results, mpvDep)

	// 4. Check Node.js / Deno
	nodeDep := Dependency{
		Name:        "Node.js",
		Required:    false,
		Description: "JavaScript runtime to solve YouTube bot challenges (highly recommended to prevent 403 Forbidden errors)",
		InstallInstructions: "sudo apt install -y nodejs (Debian/Ubuntu) or brew install node (macOS)",
	}
	if p, err := exec.LookPath("node"); err == nil {
		nodeDep.Found = true
		nodeDep.Path = p
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, p, "--version")
		if out, err := cmd.Output(); err == nil {
			nodeDep.Version = strings.TrimSpace(string(out))
		}
	} else if p, err := exec.LookPath("deno"); err == nil {
		nodeDep.Name = "Deno"
		nodeDep.Found = true
		nodeDep.Path = p
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, p, "--version")
		if out, err := cmd.Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 0 {
				nodeDep.Version = strings.TrimSpace(lines[0])
			}
		}
	}
	results = append(results, nodeDep)

	return results
}

// Helper to find yt-dlp
func resolveYtDlp() (string, error) {
	// 1. Check local bin/yt-dlp
	localPath := filepath.Join(".", "bin", "yt-dlp")
	if info, err := os.Stat(localPath); err == nil && !info.IsDir() {
		if info.Mode()&0111 != 0 {
			return localPath, nil
		}
	}

	// 2. Check system PATH
	path, err := exec.LookPath("yt-dlp")
	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("yt-dlp not found")
}
