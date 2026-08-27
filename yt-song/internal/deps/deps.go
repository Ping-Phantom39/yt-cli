package deps

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/gopxl/beep/speaker"
	"ytmusic/internal/downloader"
)

// Dependency represents a system or binary dependency
type Dependency struct {
	Name        string
	Found       bool
	Version     string
	Path        string
	Required    bool
	Description string
	InstallInstructions string
}

// CheckAll verifies all runtime dependencies of ytmusic
func CheckAll() []Dependency {
	var results []Dependency

	// 1. Check yt-dlp
	ytDlpDep := Dependency{
		Name:        "yt-dlp",
		Required:    true,
		Description: "YouTube media downloader and metadata scraper",
		InstallInstructions: "Download from https://github.com/yt-dlp/yt-dlp or place in ./bin/yt-dlp",
	}
	ytDlpPath, err := downloader.ResolvePath()
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
		Description: "Audio transcoder (converts media streams to MP3)",
		InstallInstructions: "sudo apt install -y ffmpeg  (Debian/Ubuntu) or brew install ffmpeg (macOS)",
	}
	if p, err := downloader.ResolveFFmpegPath(); err == nil {
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

	// 3. Check ffprobe
	ffprobeDep := Dependency{
		Name:        "ffprobe",
		Required:    true,
		Description: "Audio stream analyzer",
		InstallInstructions: "sudo apt install -y ffmpeg (includes ffprobe)",
	}
	if p, err := exec.LookPath("ffprobe"); err == nil {
		ffprobeDep.Found = true
		ffprobeDep.Path = p
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, p, "-version")
		if out, err := cmd.Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 0 {
				ffprobeDep.Version = strings.TrimSpace(lines[0])
			}
		}
	}
	results = append(results, ffprobeDep)

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

	// 5. Check Audio Device (ALSA/speaker)
	audioDep := Dependency{
		Name:        "Audio Output (ALSA)",
		Required:    true,
		Description: "System audio driver compatibility (libasound2 / ALSA)",
		InstallInstructions: "sudo apt install -y libasound2t64 (Ubuntu 24.04+) or libasound2 (Older Ubuntu/Debian)",
	}
	
	// Test speaker initialization
	err = speaker.Init(44100, 44100/10)
	if err == nil {
		audioDep.Found = true
		audioDep.Version = "ALSA speaker initialized successfully"
		// Clear it so it doesn't leak or block other resources
		speaker.Clear()
	} else {
		audioDep.Version = fmt.Sprintf("Error: %v", err)
	}
	results = append(results, audioDep)

	return results
}
