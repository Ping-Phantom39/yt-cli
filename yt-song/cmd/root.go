package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"ytmusic/internal/deps"
	"ytmusic/internal/ui"
)

var (
	startVolume        float64
	searchLimit        int
	cookiesFile        string
	cookiesFromBrowser string
	checkDepsFlag      bool
	startLocalMode     bool
)

var rootCmd = &cobra.Command{
	Use:   "ytmusic [query]",
	Short: "ytmusic is a cyberpunk terminal YouTube Music player",
	Long: `A high-performance cyberpunk terminal music searcher, downloader, and native audio player.
Developed in Go, leveraging yt-dlp, Bubble Tea, and gopxl/beep.

Examples:
  ytmusic                       # Start the player in search mode
  ytmusic --local               # Start directly in Local Offline Music mode
  ytmusic "lofi hip hop beats"  # Start the player and search immediately
  ytmusic -v 0.5 "synthwave"    # Start with 50% volume and search synthwave
  ytmusic --cookies-from-browser chrome "gaming lofi" # Bypass bot check using browser cookies
  ytmusic --check               # Verify system dependencies`,
	Run: func(cmd *cobra.Command, args []string) {
		// Run dependency check if flag is specified
		if checkDepsFlag {
			runDependencyCheck()
			os.Exit(0)
		}

		m := ui.NewModel()

		if startLocalMode {
			m.SetLocalMode(true)
		}

		// Setup volume override
		m.SetVolumeLevel(startVolume)

		// Auto-detect cookies.txt in the current directory if no --cookies flag was given
		if cookiesFile == "" && cookiesFromBrowser == "" {
			if _, err := os.Stat("cookies.txt"); err == nil {
				cookiesFile = "cookies.txt"
			}
		}

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

func runDependencyCheck() {
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00f0ff")).Bold(true)
	pink := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff007f")).Bold(true)
	purple := lipgloss.NewStyle().Foreground(lipgloss.Color("#7928ca")).Bold(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#39ff14")).Bold(true)
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Bold(true)
	gray := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	bold := lipgloss.NewStyle().Bold(true)

	fmt.Println(cyan.Render("=== ytmusic Cyberpunk Dependency Scanner ==="))
	fmt.Println()

	dependencies := deps.CheckAll()
	allOk := true
	hasWarnings := false

	for _, dep := range dependencies {
		var statusStr string
		if dep.Found {
			statusStr = green.Render(" [🟢 OK] ")
		} else {
			if dep.Required {
				statusStr = pink.Render(" [🔴 MISSING] ")
				allOk = false
			} else {
				statusStr = yellow.Render(" [🟡 WARNING] ")
				hasWarnings = true
			}
		}

		fmt.Printf("%s %s\n", statusStr, bold.Render(dep.Name))
		fmt.Printf("   %s: %s\n", gray.Render("Description"), dep.Description)
		if dep.Found {
			fmt.Printf("   %s: %s\n", gray.Render("Path"), dep.Path)
			if dep.Version != "" {
				fmt.Printf("   %s: %s\n", gray.Render("Version"), dep.Version)
			}
		} else {
			fmt.Printf("   %s: %s\n", pink.Render("Install Instructions"), dep.InstallInstructions)
		}
		fmt.Println()
	}

	fmt.Println(purple.Render("==========================================="))
	if !allOk {
		fmt.Println(pink.Render("❌ Error: Some required dependencies are missing."))
		fmt.Println("Please install the missing tools listed above to run the player correctly.")
	} else if hasWarnings {
		fmt.Println(yellow.Render("⚠️  Notice: Core dependencies found, but some optional tools are missing."))
		fmt.Println("The app will work, but you may face issues bypassing bot blocks without Node.js/Deno.")
	} else {
		fmt.Println(green.Render("✨ System fully configured! Ready for high-fidelity terminal playback."))
	}
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
	rootCmd.Flags().BoolVarP(&startLocalMode, "local", "m", false, "Start in Local Offline Music mode")
	rootCmd.Flags().BoolVar(&checkDepsFlag, "check", false, "Verify that all system dependencies are installed correctly")
}
