package player

import (
	"strings"
	"testing"
)

func TestBuildMpvCmd(t *testing.T) {
	p := NewPlayer()
	p.VideoOutput = "tct"

	// 1. Test local video
	cmdLocal := p.BuildMpvCmd("/tmp/test.mp4", "", "")
	argsLocal := strings.Join(cmdLocal.Args, " ")
	if !strings.Contains(argsLocal, "--vo=tct") {
		t.Errorf("Expected --vo=tct in local args, got: %s", argsLocal)
	}
	if strings.Contains(argsLocal, "--ytdl-raw-options") {
		t.Errorf("Did not expect --ytdl-raw-options for local video, got: %s", argsLocal)
	}

	// 2. Test remote YouTube video default quality
	cmdRemote := p.BuildMpvCmd("https://www.youtube.com/watch?v=lAbWm-DIB-E", "", "")
	argsRemote := strings.Join(cmdRemote.Args, " ")
	if !strings.Contains(argsRemote, "extractor-args=youtube:player_client=visionos") {
		t.Errorf("Expected player_client=visionos in remote args, got: %s", argsRemote)
	}
	if !strings.Contains(argsRemote, "--ytdl-format=bestvideo+bestaudio/best") {
		t.Errorf("Expected bestvideo+bestaudio/best format, got: %s", argsRemote)
	}
	if !strings.Contains(argsRemote, "ytdl_hook-ytdl_path=") {
		t.Errorf("Expected ytdl_hook-ytdl_path in remote args, got: %s", argsRemote)
	}

	// 3. Test remote YouTube video with custom 720p quality
	p.Quality = "720"
	cmd720 := p.BuildMpvCmd("https://www.youtube.com/watch?v=lAbWm-DIB-E", "", "")
	args720 := strings.Join(cmd720.Args, " ")
	if !strings.Contains(args720, "height<=720") {
		t.Errorf("Expected height<=720 in 720p args, got: %s", args720)
	}

	// 4. Test audio only mode
	p.Quality = "audio"
	cmdAudio := p.BuildMpvCmd("https://www.youtube.com/watch?v=lAbWm-DIB-E", "", "")
	argsAudio := strings.Join(cmdAudio.Args, " ")
	if !strings.Contains(argsAudio, "--no-video") || !strings.Contains(argsAudio, "bestaudio/best") {
		t.Errorf("Expected --no-video and bestaudio/best in audio args, got: %s", argsAudio)
	}
}
