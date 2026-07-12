package cmd

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"ytmusic/internal/ui"
)

var (
	startVolume        float64
	searchLimit        int
	cookiesFile        string
	cookiesFromBrowser string
)

var rootCmd = &cobra.Command{
	Use:   "ytmusic [query]",
	Short: "ytmusic is a cyberpunk terminal YouTube Music player",
	Long: `A high-performance cyberpunk terminal music searcher, downloader, and native audio player.
Developed in Go, leveraging yt-dlp, Bubble Tea, and gopxl/beep.

Examples:
  ytmusic                       # Start the player in search mode
  ytmusic "lofi hip hop beats"  # Start the player and search immediately
  ytmusic -v 0.5 "synthwave"    # Start with 50% volume and search synthwave
  ytmusic --cookies-from-browser chrome "gaming lofi" # Bypass bot check using browser cookies`,
	Run: func(cmd *cobra.Command, args []string) {
		m := ui.NewModel()

		// Setup volume override
		m.SetVolumeLevel(startVolume)

		// Setup cookies parameters
		m.SetCookies(cookiesFile, cookiesFromBrowser)

		// Setup boot search query if provided as arguments
		if len(args) > 0 {
			query := strings.Join(args, " ")
			m.SetInitialSearch(query)
		}

		// Run Bubble Tea application in full-screen alt-buffer
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Fatal runtime TUI error: %v\n", err)
			os.Exit(1)
		}
	},
}

// Execute triggers the Cobra command execution pipeline
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Command execution error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().Float64VarP(&startVolume, "volume", "v", 0.8, "Starting volume level (0.0 to 1.0)")
	rootCmd.Flags().IntVarP(&searchLimit, "limit", "l", 15, "Number of search results to fetch")
	rootCmd.Flags().StringVar(&cookiesFile, "cookies", "", "Path to cookies file")
	rootCmd.Flags().StringVar(&cookiesFromBrowser, "cookies-from-browser", "", "Load cookies from a specific browser (e.g. chrome, firefox, edge, brave)")
}
