package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"ytplayer/internal/deps"
	"ytplayer/internal/ui"
)

var (
	searchLimit        int
	cookiesFile        string
	cookiesFromBrowser string
	videoOutput        string
	quality            string
	checkDepsFlag      bool
	startLocalMode     bool
)

var rootCmd = &cobra.Command{
	Use:   "ytplayer [query]",
	Short: "ytplayer is a cyberpunk terminal YouTube video player and downloader",
	Long: `A high-performance cyberpunk terminal video searcher, downloader, and mpv-backed streaming player.
Developed in Go, leveraging yt-dlp, Bubble Tea, and mpv.

Examples:
  ytplayer                       # Start the player in search mode
  ytplayer --local               # Start directly in Local Offline Video mode
  ytplayer "cyberpunk synthwave" # Start the player and search immediately
  ytplayer --vo tct "lofi beats" # Render video in terminal using true-color text (headless)
  ytplayer --cookies cookies.txt # Bypass bot check using a cookies file
  ytplayer --check               # Verify system dependencies`,
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

		// Auto-detect cookies.txt in the current directory if no --cookies flag was given
		if cookiesFile == "" && cookiesFromBrowser == "" {
			if _, err := os.Stat("cookies.txt"); err == nil {
				cookiesFile = "cookies.txt"
			}
		}

		// Setup cookies parameters
		m.SetCookies(cookiesFile, cookiesFromBrowser)

		// Setup video output override
		m.SetVideoOutput(videoOutput)

		// Setup stream quality setting
		m.SetQuality(quality)

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

	fmt.Println(cyan.Render("=== ytplayer Cyberpunk Dependency Scanner ==="))
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
	rootCmd.Flags().IntVarP(&searchLimit, "limit", "l", 15, "Number of search results to fetch")
	rootCmd.Flags().StringVar(&cookiesFile, "cookies", "", "Path to cookies file")
	rootCmd.Flags().StringVar(&cookiesFromBrowser, "cookies-from-browser", "", "Load cookies from a specific browser (e.g. chrome, firefox, edge, brave)")
	rootCmd.Flags().StringVar(&videoOutput, "vo", "", "Force custom mpv video output driver (e.g. tct, sixel, kitty, gpu)")
	rootCmd.Flags().StringVarP(&quality, "quality", "q", "best", "Max playback quality (e.g. best, 1080, 720, 480, 360)")
	rootCmd.Flags().BoolVarP(&startLocalMode, "local", "m", false, "Start in Local Offline Video mode")
	rootCmd.Flags().BoolVar(&checkDepsFlag, "check", false, "Verify that all system dependencies are installed correctly")
}
