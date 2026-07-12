package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"ytmusic/internal/downloader"
	"ytmusic/internal/player"
)

// Message types for Bubble Tea event loop
type searchResultMsg struct {
	videos []downloader.Video
	err    error
}

type progressMsg float64
type downloadFinishedMsg string
type downloadErrorMsg error
type tickMsg time.Time
type trackFinishedMsg struct{}

// Style definitions
var (
	cyanColor   = lipgloss.Color("#00f0ff")  // Cyberpunk Neon Cyan
	pinkColor   = lipgloss.Color("#ff007f")  // Cyberpunk Neon Pink
	purpleColor = lipgloss.Color("#7928ca")  // Dark Cyberpunk Purple
	grayColor   = lipgloss.Color("#666666")  // Muted grey
	darkGrayBg  = lipgloss.Color("#121212")  // Terminal slate background
	greenColor  = lipgloss.Color("#39ff14")  // Radioactive Green
	yellowColor = lipgloss.Color("#ffff00")  // Safety Yellow

	// Title / Banner Styling
	titleStyle = lipgloss.NewStyle().
			Foreground(cyanColor).
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(purpleColor).
			MarginBottom(1)

	// List styling
	selectedRowStyle = lipgloss.NewStyle().
				Foreground(pinkColor).
				Background(lipgloss.Color("#1a0917")).
				Bold(true)

	normalRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff"))

	metaStyle = lipgloss.NewStyle().
			Foreground(grayColor)

	// Status styles
	greenStatusStyle = lipgloss.NewStyle().
				Foreground(greenColor).
				Bold(true)

	yellowStatusStyle = lipgloss.NewStyle().
				Foreground(yellowColor).
				Bold(true)

	redStatusStyle = lipgloss.NewStyle().
			Foreground(pinkColor).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(cyanColor).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(pinkColor).
			Bold(true)

	statusStoppedStyle = lipgloss.NewStyle().
			Foreground(grayColor).
			Bold(true)

	searchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cyanColor).
			Padding(0, 1)

	resultsBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purpleColor).
			Padding(0, 1)

	playerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(pinkColor).
			Padding(1, 2)
)

// Model represents the state of the TUI application
type Model struct {
	// Search state
	searchInput textinput.Model
	searching   bool
	searchErr   error

	// Results list
	results []downloader.Video
	cursor  int // index of highlighted row

	// Downloading state
	downloading     bool
	downloadPercent float64
	downloadVideoID string
	downloadErr     error
	progressChan    chan float64
	downloadCancel  context.CancelFunc

	// Player state
	player            *player.Player
	playingVideo      downloader.Video
	currentPos        float64
	totalDuration     float64
	volumeLevel       float64
	playState         player.PlayState
	trackFinishedChan chan struct{}

	// Cookies config
	CookiesFile        string
	CookiesFromBrowser string

	// Terminal constraints
	width  int
	height int
}

// NewModel builds and returns the initial TUI model
func NewModel() *Model {
	ti := textinput.New()
	ti.Placeholder = "Enter artist, song, or video title..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40
	ti.Prompt = "⚡ Search: "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(cyanColor).Bold(true)

	p := player.NewPlayer()
	finishedChan := make(chan struct{}, 1)
	p.SetOnFinish(func() {
		select {
		case finishedChan <- struct{}{}:
		default:
		}
	})

	return &Model{
		searchInput:       ti,
		player:            p,
		trackFinishedChan: finishedChan,
		volumeLevel:       0.8,
	}
}

// SetInitialSearch configures a search query to run automatically on startup.
func (m *Model) SetInitialSearch(query string) {
	m.searchInput.SetValue(query)
	m.searching = true
	m.searchErr = nil
	m.searchInput.Blur()
}

// SetVolumeLevel overrides the default starting volume level.
func (m *Model) SetVolumeLevel(vol float64) {
	if vol < 0 {
		vol = 0
	}
	if vol > 1 {
		vol = 1
	}
	m.volumeLevel = vol
	m.player.SetVolume(vol)
}

// SetCookies configures YouTube authentication options.
func (m *Model) SetCookies(cookiesFile, cookiesFromBrowser string) {
	m.CookiesFile = cookiesFile
	m.CookiesFromBrowser = cookiesFromBrowser
}

// Init initializes the Bubble Tea model (sets up initial commands)
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		textinput.Blink,
		tickCmd(),
		waitForTrackFinished(m.trackFinishedChan),
	}

	if m.searching && m.searchInput.Value() != "" {
		cmds = append(cmds, m.searchCmd(context.Background(), m.searchInput.Value()))
	}

	return tea.Batch(cmds...)
}

// Commands used in the model
func (m *Model) searchCmd(ctx context.Context, query string) tea.Cmd {
	return func() tea.Msg {
		videos, err := downloader.Search(ctx, query, 15, m.CookiesFile, m.CookiesFromBrowser)
		return searchResultMsg{videos: videos, err: err}
	}
}

func (m *Model) downloadCmd(ctx context.Context, id string, progressChan chan float64) tea.Cmd {
	return func() tea.Msg {
		// Close channel upon completion so reader finishes
		defer func() {
			recover() // Ignore panic if already closed
		}()

		filePath, err := downloader.Download(ctx, id, progressChan, m.CookiesFile, m.CookiesFromBrowser)
		if err != nil {
			return downloadErrorMsg(err)
		}
		return downloadFinishedMsg(filePath)
	}
}

func waitForProgress(ch <-chan float64) tea.Cmd {
	return func() tea.Msg {
		p, ok := <-ch
		if !ok {
			return nil
		}
		return progressMsg(p)
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*250, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func waitForTrackFinished(ch <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-ch
		return trackFinishedMsg{}
	}
}

// Helper to launch download and play sequence
func (m *Model) startDownloadAndPlay(video downloader.Video) tea.Cmd {
	// Cancel any active download
	if m.downloadCancel != nil {
		m.downloadCancel()
	}

	m.downloading = true
	m.downloadPercent = 0.0
	m.downloadVideoID = video.ID
	m.downloadErr = nil
	m.progressChan = make(chan float64, 50)
	m.playingVideo = video

	var ctx context.Context
	ctx, m.downloadCancel = context.WithCancel(context.Background())

	return tea.Batch(
		m.downloadCmd(ctx, video.ID, m.progressChan),
		waitForProgress(m.progressChan),
	)
}

// Update handles interactive keyboard inputs and async events
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.searchInput.Width = m.width - 16
		return m, nil

	case tickMsg:
		// Query the player's runtime metrics
		m.currentPos, m.totalDuration, m.volumeLevel, m.playState = m.player.Status()
		return m, tickCmd()

	case trackFinishedMsg:
		// Song ended. Listen for next track finish first
		nextTrackCmd := waitForTrackFinished(m.trackFinishedChan)

		// Implement auto-play: Find if there's a next video in the list
		if len(m.results) > 0 {
			currentIndex := -1
			for i, v := range m.results {
				if v.ID == m.playingVideo.ID {
					currentIndex = i
					break
				}
			}

			nextIndex := currentIndex + 1
			if nextIndex < len(m.results) {
				m.cursor = nextIndex
				m.playingVideo = m.results[nextIndex]
				playCmd := m.startDownloadAndPlay(m.results[nextIndex])
				return m, tea.Batch(nextTrackCmd, playCmd)
			}
		}

		m.playingVideo = downloader.Video{}
		return m, nextTrackCmd

	case searchResultMsg:
		m.searching = false
		if msg.err != nil {
			m.searchErr = msg.err
			m.results = nil
		} else {
			m.results = msg.videos
			m.searchErr = nil
			m.cursor = 0
		}
		return m, nil

	case progressMsg:
		m.downloadPercent = float64(msg)
		// Re-schedule listening on the channel for subsequent updates
		if m.downloading {
			return m, waitForProgress(m.progressChan)
		}
		return m, nil

	case downloadFinishedMsg:
		m.downloading = false
		m.downloadPercent = 100.0
		// Play downloaded path
		err := m.player.Play(string(msg))
		if err != nil {
			m.downloadErr = fmt.Errorf("playback failed: %v", err)
		}
		return m, nil

	case downloadErrorMsg:
		// Check if it was canceled manually
		if ctxErr := context.Canceled; msg.Error() == ctxErr.Error() || strings.Contains(msg.Error(), "context canceled") {
			return m, nil
		}
		m.downloading = false
		m.downloadErr = msg
		return m, nil

	case tea.KeyMsg:
		// Handle global keys when search input is NOT focused
		if !m.searchInput.Focused() {
			switch msg.String() {
			case "q", "ctrl+c":
				m.player.Stop()
				if m.downloadCancel != nil {
					m.downloadCancel()
				}
				return m, tea.Quit

			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}

			case "down", "j":
				if m.cursor < len(m.results)-1 {
					m.cursor++
				}

			case "space":
				m.player.TogglePause()

			case "/":
				m.searchInput.Focus()
				return m, textinput.Blink

			case "enter":
				if len(m.results) > 0 && m.cursor < len(m.results) {
					playCmd := m.startDownloadAndPlay(m.results[m.cursor])
					return m, playCmd
				}

			case "left", "h":
				// Seek backward 5 seconds
				if m.playState != player.StateStopped {
					newPos := m.currentPos - 5.0
					_ = m.player.Seek(newPos)
				}

			case "right", "l":
				// Seek forward 5 seconds
				if m.playState != player.StateStopped {
					newPos := m.currentPos + 5.0
					_ = m.player.Seek(newPos)
				}

			case "[":
				// Volume down
				m.player.SetVolume(m.volumeLevel - 0.05)

			case "]":
				// Volume up
				m.player.SetVolume(m.volumeLevel + 0.05)

			case "s":
				// Stop playback
				m.player.Stop()
			}
		} else {
			// Search Input is focused
			switch msg.String() {
			case "enter":
				query := m.searchInput.Value()
				if strings.TrimSpace(query) != "" {
					m.searching = true
					m.searchErr = nil
					m.searchInput.Blur()
					return m, m.searchCmd(context.Background(), query)
				}

			case "esc":
				m.searchInput.Blur()
			}

			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// formatTime formats seconds as MM:SS
func formatTime(seconds float64) string {
	s := int(seconds)
	if s < 0 {
		s = 0
	}
	m := s / 60
	sec := s % 60
	return fmt.Sprintf("%02d:%02d", m, sec)
}// renderCustomProgressBar builds a cyberpunk progress bar using ascii characters
func renderCustomProgressBar(width int, percent float64, color lipgloss.Color) string {
	if width <= 0 {
		return ""
	}
	filledWidth := int(percent * float64(width))
	if filledWidth < 0 {
		filledWidth = 0
	}
	if filledWidth > width {
		filledWidth = width
	}

	emptyWidth := width - filledWidth

	filledStr := strings.Repeat("█", filledWidth)
	emptyStr := strings.Repeat("░", emptyWidth)

	// Style sections
	filledStyled := lipgloss.NewStyle().Foreground(color).Render(filledStr)
	emptyStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("#252525")).Render(emptyStr)

	return filledStyled + emptyStyled
}

// truncate helper to keep UI alignment within terminal boundary
func truncate(s string, length int) string {
	if len(s) > length {
		if length > 3 {
			return s[:length-3] + "..."
		}
		return s[:length]
	}
	return s
}

// View renders the TUI layout on every update cycle
func (m *Model) View() string {
	if m.width < 40 || m.height < 15 {
		return fmt.Sprintf("Terminal too small (%dx%d).\nPlease enlarge to at least 40x15.", m.width, m.height)
	}

	var s strings.Builder

	// Header Banner
	banner := `
 █ █ █▀▀█ █  █ ▀▀█▀▀ █  █ █▀▀█ █▀▀▀    █▀▄▀█ █  █ █▀▀▀ █▀▀▀█ █▀▀▀
 ▀▄▀ █  █ █  █   █   █  █ █▀▀▄ █▀▀▀    █ ▀ █ █  █ ▀▀▀█ █▄▄▄█ █▀▀▀
  █  ▀▀▀▀  ▀▀▀   ▀   ▀▀▀▀ █▄▄█ █▄▄▄    █   █  ▀▀▀ █▄▄▄ █    ▀ █▄▄▄
`
	s.WriteString(lipgloss.NewStyle().Foreground(cyanColor).Align(lipgloss.Center).Width(m.width).Render(banner))
	s.WriteString("\n")

	// Search bar view
	searchContent := m.searchInput.View()
	s.WriteString(searchBoxStyle.Width(m.width - 4).Render(searchContent))
	s.WriteString("\n")

	// Main body height limits
	mainContentHeight := m.height - 18
	if mainContentHeight < 3 {
		mainContentHeight = 3
	}

	innerWidth := m.width - 8
	maxChanLen := 20
	maxTitleLen := innerWidth - maxChanLen - 12
	if maxTitleLen < 20 {
		maxTitleLen = 20
	}

	// Main List area (within results box)
	var resultsContent string
	if m.searching {
		resultsContent = "\n" + lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(infoStyle.Render("⚡ Scraping YouTube metadata asynchronously...\n\nPlease wait a moment.")) + "\n"
	} else if m.searchErr != nil {
		resultsContent = "\n" + lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(errorStyle.Render(fmt.Sprintf("⚠️ Search Failed:\n\n%v", m.searchErr))) + "\n"
	} else if len(m.results) == 0 {
		resultsContent = "\n" + lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(infoStyle.Render("No search results found.\n\nPress [/] to focus the search bar and type a query.")) + "\n"
	} else {
		var sb strings.Builder
		headerStr := fmt.Sprintf("  %-*s  %-*s  %6s", maxTitleLen, "TITLE", maxChanLen, "UPLOADER", "DURATION")
		headerStyled := lipgloss.NewStyle().Foreground(purpleColor).Bold(true).Underline(true).Render(headerStr)
		sb.WriteString(headerStyled + "\n\n")

		for i, video := range m.results {
			if i >= mainContentHeight {
				break
			}
			title := truncate(video.Title, maxTitleLen)
			uploader := truncate(video.Uploader, maxChanLen)
			duration := video.FormatDuration()

			var rowStr string
			if i == m.cursor {
				rawRow := fmt.Sprintf("▶ %-*s  %-*s  %6s", maxTitleLen, title, maxChanLen, uploader, duration)
				// Pad rawRow to fill the innerWidth minus padding
				rawRow = fmt.Sprintf("%-*s", innerWidth-2, rawRow)
				rowStr = selectedRowStyle.Render(rawRow)
			} else {
				titleStyled := normalRowStyle.Render(title)
				uploaderStyled := metaStyle.Render(uploader)
				durationStyled := lipgloss.NewStyle().Foreground(yellowColor).Render(duration)
				rowStr = fmt.Sprintf("  %-*s  %-*s  %6s", 
					maxTitleLen, titleStyled, 
					maxChanLen, uploaderStyled, 
					durationStyled,
				)
			}
			sb.WriteString(rowStr + "\n")
		}
		resultsContent = sb.String()
	}

	s.WriteString(resultsBoxStyle.Width(m.width - 4).Render(resultsContent))
	s.WriteString("\n")

	// Player Panel Section
	var playerContent strings.Builder
	
	if m.downloading {
		badge := lipgloss.NewStyle().Background(pinkColor).Foreground(darkGrayBg).Bold(true).Padding(0, 1).Render("📥 DOWNLOADING")
		titleStyled := lipgloss.NewStyle().Foreground(pinkColor).Bold(true).Render(truncate(m.playingVideo.Title, m.width-25))
		playerContent.WriteString(fmt.Sprintf("%s  %s\n\n", badge, titleStyled))

		progBarWidth := m.width - 32
		if progBarWidth < 10 {
			progBarWidth = 10
		}
		
		timelineStr := fmt.Sprintf("Progress: [%s] %5.1f%%  │  Transcoding MP3...",
			renderCustomProgressBar(progBarWidth, m.downloadPercent/100.0, pinkColor),
			m.downloadPercent,
		)
		playerContent.WriteString(timelineStr)
	} else if m.downloadErr != nil {
		badge := lipgloss.NewStyle().Background(lipgloss.Color("#ff007f")).Foreground(darkGrayBg).Bold(true).Padding(0, 1).Render("⚠️ ERROR")
		errStyled := errorStyle.Render(fmt.Sprintf("Download failed: %v", m.downloadErr))
		playerContent.WriteString(fmt.Sprintf("%s  %s\n\n", badge, errStyled))
		playerContent.WriteString(lipgloss.NewStyle().Foreground(grayColor).Render("Press Enter on a search result to try again."))
	} else if m.playState != player.StateStopped {
		badgeText := "▶ PLAYING"
		var badgeBg lipgloss.Color = greenColor
		if m.playState == player.StatePaused {
			badgeText = "⏸ PAUSED"
			badgeBg = yellowColor
		}
		
		badge := lipgloss.NewStyle().Background(badgeBg).Foreground(darkGrayBg).Bold(true).Padding(0, 1).Render(badgeText)
		titleStyled := lipgloss.NewStyle().Foreground(cyanColor).Bold(true).Render(truncate(m.playingVideo.Title, m.width-25))
		playerContent.WriteString(fmt.Sprintf("%s  %s\n\n", badge, titleStyled))

		timelineWidth := m.width - 32
		if timelineWidth < 10 {
			timelineWidth = 10
		}
		var timelinePercent float64
		if m.totalDuration > 0 {
			timelinePercent = m.currentPos / m.totalDuration
		}

		volPercent := int(m.volumeLevel * 100)
		volIcon := "🔊"
		if volPercent == 0 {
			volIcon = "🔇"
		} else if volPercent < 40 {
			volIcon = "🔈"
		} else if volPercent < 85 {
			volIcon = "🔉"
		}

		timelineStr := fmt.Sprintf("%5s [%s] %-5s  │  %s %3d%%",
			formatTime(m.currentPos),
			renderCustomProgressBar(timelineWidth, timelinePercent, cyanColor),
			formatTime(m.totalDuration),
			volIcon,
			volPercent,
		)
		playerContent.WriteString(timelineStr)
	} else {
		badge := lipgloss.NewStyle().Background(grayColor).Foreground(darkGrayBg).Bold(true).Padding(0, 1).Render("■ STOPPED")
		titleStyled := lipgloss.NewStyle().Foreground(grayColor).Render("No active playback")
		playerContent.WriteString(fmt.Sprintf("%s  %s\n\n", badge, titleStyled))
		playerContent.WriteString(lipgloss.NewStyle().Foreground(grayColor).Render("Select a song from the results and press Enter to stream."))
	}

	s.WriteString(playerBoxStyle.Width(m.width - 4).Render(playerContent.String()))
	s.WriteString("\n")

	// Help Guide / Footer
	keyStyle := lipgloss.NewStyle().Foreground(cyanColor).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(grayColor)
	
	helpStr := fmt.Sprintf(
		" %s %s   %s %s   %s %s   %s %s   %s %s   %s %s   %s %s",
		keyStyle.Render("[/]"), descStyle.Render("Search"),
		keyStyle.Render("[Enter]"), descStyle.Render("Play"),
		keyStyle.Render("[Space]"), descStyle.Render("Pause/Resume"),
		keyStyle.Render("[s]"), descStyle.Render("Stop"),
		keyStyle.Render("[←/→]"), descStyle.Render("Seek ±5s"),
		keyStyle.Render("[[/]]"), descStyle.Render("Vol -/+"),
		keyStyle.Render("[q]"), descStyle.Render("Quit"),
	)
	s.WriteString(lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(helpStr))

	return s.String()
}
