package downloader

import (
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Cyberpunk 2077 - Official Trailer",
			expected: "Cyberpunk 2077 - Official Trailer",
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
			expected: "video",
		},
	}

	for _, tt := range tests {
		got := SanitizeFilename(tt.input)
		if got != tt.expected {
			t.Errorf("SanitizeFilename(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}
