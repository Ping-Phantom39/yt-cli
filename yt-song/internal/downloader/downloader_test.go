package downloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Cyberpunk 2077 - Official Soundtrack",
			expected: "Cyberpunk 2077 - Official Soundtrack",
		},
		{
			input:    "What is API? | System Design",
			expected: "What is API_System Design",
		},
		{
			input:    "How to Code: Step-by-Step / Full Guide",
			expected: "How to Code_Step-by-Step_Full Guide",
		},
		{
			input:    `Illegal: \ / : * ? " < > | Chars`,
			expected: "Illegal_Chars",
		},
		{
			input:    "   ...Leading and Trailing Dots/Spaces...   ",
			expected: "Leading and Trailing Dots_Spaces",
		},
		{
			input:    "???",
			expected: "song",
		},
	}

	for _, tt := range tests {
		got := SanitizeFilename(tt.input)
		if got != tt.expected {
			t.Errorf("SanitizeFilename(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestScanLocalMusic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "testmusic-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test dummy audio files
	testFile1 := filepath.Join(tempDir, "song1.mp3")
	testFile2 := filepath.Join(tempDir, "song2.flac")
	dummyText := filepath.Join(tempDir, "ignore.txt")

	_ = os.WriteFile(testFile1, []byte("dummy mp3"), 0644)
	_ = os.WriteFile(testFile2, []byte("dummy flac"), 0644)
	_ = os.WriteFile(dummyText, []byte("ignore text"), 0644)

	videos, err := ScanLocalMusic(tempDir)
	if err != nil {
		t.Fatalf("ScanLocalMusic returned error: %v", err)
	}

	if len(videos) != 2 {
		t.Errorf("Expected 2 audio files, got %d", len(videos))
	}

	foundSong1 := false
	foundSong2 := false

	for _, v := range videos {
		if v.Title == "song1" {
			foundSong1 = true
		}
		if v.Title == "song2" {
			foundSong2 = true
		}
	}

	if !foundSong1 || !foundSong2 {
		t.Errorf("ScanLocalMusic failed to detect test audio files. Got: %+v", videos)
	}
}
