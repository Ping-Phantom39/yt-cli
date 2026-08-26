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

	// 2. Test remote YouTube video
	cmdRemote := p.BuildMpvCmd("https://www.youtube.com/watch?v=lAbWm-DIB-E", "", "")
	argsRemote := strings.Join(cmdRemote.Args, " ")
	if !strings.Contains(argsRemote, "extractor-args=youtube:player_client=android") {
		t.Errorf("Expected player_client=android in remote args, got: %s", argsRemote)
	}
	if !strings.Contains(argsRemote, "ytdl_hook-ytdl_path=") {
		t.Errorf("Expected ytdl_hook-ytdl_path in remote args, got: %s", argsRemote)
	}
}
