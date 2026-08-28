package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Ping-Phantom39/yt-cli/yt-player/internal/downloader"
	"github.com/Ping-Phantom39/yt-cli/yt-player/internal/player"
)

// Message types for Bubble Tea event loop
type searchResultMsg struct {
	videos []downloader.Video
	err    error
}

type progressMsg float64
type downloadFinishedMsg string
type downloadErrorMsg error
type playFinishedMsg struct {
	err error
}

// Style definitions
var (
	cyanColor   = lipgloss.Color("#00f0ff")  // Cyberpunk Neon Cyan
	pinkColor   = lipgloss.Color("#ff007f")  // Cyberpunk Neon Pink
	purpleColor = lipgloss.Color("#7928ca")  // Dark Cyberpunk Purple
	grayColor   = lipgloss.Color("#666666")  // Muted grey
	darkGrayBg  = lipgloss.Color("#121212")  // Terminal slate background
	greenColor  = lipgloss.Color("#39ff14")  // Radioactive Green
	yellowColor = lipgloss.Color("#ffff00")  // Safety Yellow

	titleStyle = lipgloss.NewStyle().
			Foreground(cyanColor).
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(purpleColor).
			MarginBottom(1)

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(pinkColor).
				Background(lipgloss.Color("#1a0917")).
				Bold(true)

	normalRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff"))

	metaStyle = lipgloss.NewStyle().
			Foreground(grayColor)

	greenStatusStyle = lipgloss.NewStyle().
				Foreground(greenColor).
				Bold(true)

	yellowStatusStyle = lipgloss.NewStyle().
				Foreground(yellowColor).
				Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(cyanColor).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(pinkColor).
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
	results  []downloader.Video
	cursor   int // index of highlighted row
	startIdx int // index of first visible row in viewport

	// Downloading state
	downloading         bool
	downloadPercent     float64
	downloadVideoID     string
	downloadErr         error
	progressChan        chan float64
	downloadCancel      context.CancelFunc
	isPermanentDownload bool
	downloadSuccessMsg  string

	// Playback State (last status)
	playbackErr error

	// Cookies config
	CookiesFile        string
	CookiesFromBrowser string

	// Player config
	player *player.Player

	// Video Quality Preset State
	selectedQualityIndex int

	// Local Video state
	isLocalMode bool
	localVideos []downloader.Video

	// Terminal constraints
	width  int
	height int
}

// NewModel builds and returns the initial TUI model
func NewModel() *Model {
	ti := textinput.New()
	ti.Placeholder = "Enter keywords or search phrase..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40
	ti.Prompt = "⚡ Search: "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(cyanColor).Bold(true)

	p := player.NewPlayer()

	return &Model{
		searchInput:          ti,
		player:               p,
		selectedQualityIndex: 0, // Default to Best / 4K
	}
}

// SetInitialSearch configures a search query to run automatically on startup.
func (m *Model) SetInitialSearch(query string) {
	m.searchInput.SetValue(query)
	m.searching = true
	m.searchErr = nil
	m.searchInput.Blur()
}

// SetCookies configures YouTube authentication options.
func (m *Model) SetCookies(cookiesFile, cookiesFromBrowser string) {
	m.CookiesFile = cookiesFile
	m.CookiesFromBrowser = cookiesFromBrowser
}

// SetVideoOutput overrides the video output driver
func (m *Model) SetVideoOutput(vo string) {
	m.player.VideoOutput = vo
}

// SetQuality sets the video stream quality limit (e.g. 1080, 720, 480, best, audio)
func (m *Model) SetQuality(q string) {
	lower := strings.ToLower(strings.TrimSpace(q))
	for idx, preset := range player.QualityPresets {
		if preset.ID == lower || strings.HasPrefix(lower, preset.ID) {
			m.selectedQualityIndex = idx
			m.player.Quality = preset.ID
			return
		}
	}
	m.player.Quality = q
}

// CycleQuality cycles forward to the next quality preset
func (m *Model) CycleQuality() {
	m.selectedQualityIndex = (m.selectedQualityIndex + 1) % len(player.QualityPresets)
	m.player.Quality = player.QualityPresets[m.selectedQualityIndex].ID
}

// SetQualityByIndex sets the quality preset by its numeric index
func (m *Model) SetQualityByIndex(idx int) {
	if idx >= 0 && idx < len(player.QualityPresets) {
		m.selectedQualityIndex = idx
		m.player.Quality = player.QualityPresets[idx].ID
	}
}

// SetLocalMode configures whether the TUI starts in Local Offline mode or Online YouTube mode.
func (m *Model) SetLocalMode(local bool) {
	m.isLocalMode = local
	if local {
		m.searchInput.Placeholder = "Filter local videos..."
		m.loadLocalVideos()
	} else {
		m.searchInput.Placeholder = "Enter keywords or search phrase..."
	}
}

func (m *Model) loadLocalVideos() {
	videos, _ := downloader.ScanLocalVideos(".", "./downloads")
	m.localVideos = videos
	m.filterLocalResults()
}

func (m *Model) filterLocalResults() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
	if query == "" {
		m.results = m.localVideos
	} else {
		var filtered []downloader.Video
		for _, v := range m.localVideos {
			if strings.Contains(strings.ToLower(v.Title), query) || strings.Contains(strings.ToLower(v.Uploader), query) {
				filtered = append(filtered, v)
			}
		}
		m.results = filtered
	}
	m.cursor = 0
	m.startIdx = 0
}

// Init initializes the Bubble Tea model
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		textinput.Blink,
	}

	if m.isLocalMode {
		m.loadLocalVideos()
	} else if m.searching && m.searchInput.Value() != "" {
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

func (m *Model) downloadCmd(ctx context.Context, id string, outputPath string, quality string, progressChan chan float64) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			recover() // Ignore panic if already closed
		}()

		filePath, err := downloader.Download(ctx, id, outputPath, quality, progressChan, m.CookiesFile, m.CookiesFromBrowser)
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

// Update handles interactive keyboard inputs and async events
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.searchInput.Width = m.width - 16
		return m, nil

	case searchResultMsg:
		m.searching = false
		if msg.err != nil {
			m.searchErr = msg.err
			m.results = nil
		} else {
			m.results = msg.videos
			m.searchErr = nil
			m.cursor = 0
			m.startIdx = 0
		}
		return m, nil

	case progressMsg:
		m.downloadPercent = float64(msg)
		if m.downloading {
			return m, waitForProgress(m.progressChan)
		}
		return m, nil

	case downloadFinishedMsg:
		m.downloading = false
		m.downloadPercent = 100.0
		m.downloadSuccessMsg = fmt.Sprintf("Saved permanently to downloads/%s", filepath.Base(string(msg)))
		return m, nil

	case downloadErrorMsg:
		if ctxErr := context.Canceled; msg.Error() == ctxErr.Error() || strings.Contains(msg.Error(), "context canceled") {
			return m, nil
		}
		m.downloading = false
		errMsg := msg.Error()
		if strings.Contains(errMsg, "confirm you’re not a bot") || strings.Contains(errMsg, "confirm you're not a bot") || strings.Contains(errMsg, "Sign in to confirm") {
			m.downloadErr = fmt.Errorf("YouTube bot block! Bypassed by passing cookies (e.g. --cookies cookies.txt or --cookies-from-browser chrome)")
		} else {
			m.downloadErr = msg
		}
		return m, nil

	case playFinishedMsg:
		// Play finished. If there was an error running mpv, record it
		m.playbackErr = msg.err
		return m, nil

	case tea.KeyMsg:
		if !m.searchInput.Focused() {
			switch msg.String() {
			case "q", "ctrl+c":
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

			case "/":
				m.searchInput.Focus()
				m.playbackErr = nil
				return m, textinput.Blink

			case "m":
				m.isLocalMode = !m.isLocalMode
				m.playbackErr = nil
				m.downloadErr = nil
				m.searchErr = nil
				m.searching = false
				if m.isLocalMode {
					m.searchInput.Placeholder = "Filter local videos..."
					m.loadLocalVideos()
				} else {
					m.searchInput.Placeholder = "Enter keywords or search phrase..."
					m.results = nil
				}
				return m, nil

			case "v", "tab":
				m.CycleQuality()
				return m, nil

			case "1":
				m.SetQualityByIndex(0)
				return m, nil

			case "2":
				m.SetQualityByIndex(1)
				return m, nil

			case "3":
				m.SetQualityByIndex(2)
				return m, nil

			case "4":
				m.SetQualityByIndex(3)
				return m, nil

			case "5":
				m.SetQualityByIndex(4)
				return m, nil

			case "6":
				m.SetQualityByIndex(5)
				return m, nil

			case "enter":
				if len(m.results) > 0 && m.cursor < len(m.results) {
					m.playbackErr = nil
					video := m.results[m.cursor]

					// Local file: play directly via mpv
					if strings.HasPrefix(video.ID, "local:") {
						c := m.player.BuildMpvCmd(video.URL, "", "")
						return m, tea.ExecProcess(c, func(err error) tea.Msg {
							return playFinishedMsg{err: err}
						})
					}

					// Check if a permanent download file already exists in ./downloads
					titleFileName := fmt.Sprintf("%s.mp4", downloader.SanitizeFilename(video.Title))
					titlePath := filepath.Join(".", "downloads", titleFileName)
					idPath := filepath.Join(".", "downloads", fmt.Sprintf("%s.mp4", video.ID))

					var target string
					if info, err := os.Stat(titlePath); err == nil && !info.IsDir() && info.Size() > 0 {
						target = titlePath
					} else if info, err := os.Stat(idPath); err == nil && !info.IsDir() && info.Size() > 0 {
						target = idPath
					} else {
						target = "https://www.youtube.com/watch?v=" + video.ID
					}

					// Suspend TUI and play via mpv
					c := m.player.BuildMpvCmd(target, m.CookiesFile, m.CookiesFromBrowser)
					return m, tea.ExecProcess(c, func(err error) tea.Msg {
						return playFinishedMsg{err: err}
					})
				}

			case "d":
				if len(m.results) > 0 && m.cursor < len(m.results) {
					m.playbackErr = nil
					video := m.results[m.cursor]

					if m.downloadCancel != nil {
						m.downloadCancel()
					}

					m.downloading = true
					m.downloadPercent = 0.0
					m.downloadVideoID = video.ID
					m.downloadErr = nil
					m.downloadSuccessMsg = ""
					m.progressChan = make(chan float64, 50)

					var ctx context.Context
					ctx, m.downloadCancel = context.WithCancel(context.Background())

					filename := fmt.Sprintf("%s.mp4", downloader.SanitizeFilename(video.Title))
					outputPath := filepath.Join(".", "downloads", filename)

					return m, tea.Batch(
						m.downloadCmd(ctx, video.ID, outputPath, m.player.Quality, m.progressChan),
						waitForProgress(m.progressChan),
					)
				}
			}
		} else {
			// Search Input is focused
			switch msg.String() {
			case "enter":
				query := m.searchInput.Value()
				if m.isLocalMode {
					m.filterLocalResults()
					m.searchInput.Blur()
					return m, nil
				}
				if strings.TrimSpace(query) != "" {
					m.searching = true
					m.searchErr = nil
					m.cursor = 0
					m.startIdx = 0
					m.searchInput.Blur()
					return m, m.searchCmd(context.Background(), query)
				}

			case "esc":
				m.searchInput.Blur()
			}

			m.searchInput, cmd = m.searchInput.Update(msg)
			if m.isLocalMode {
				m.filterLocalResults()
			}
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
}

// renderCustomProgressBar builds a cyberpunk progress bar using ascii characters
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
  █ █ █▀▀█ ▀▀█▀▀ █▀▀█ █    █▀▀█ █  █ █▀▀▀ █▀▀█ 
  ▀▄▀ █  █   █   █▄▄█ █    █▄▄█ ▀▄▄█ █▀▀▀ █▄▄▀ 
   ▀  ▀▀▀▀   ▀   █    █▄▄█ ▀  ▀ ▄▄▄█ █▄▄▄ ▀ ▀▀ 
`
	s.WriteString(lipgloss.NewStyle().Foreground(cyanColor).Align(lipgloss.Center).Width(m.width).Render(banner))
	s.WriteString("\n")

	// Mode & Quality Badges
	var modeBadge string
	if m.isLocalMode {
		modeBadge = lipgloss.NewStyle().Background(greenColor).Foreground(darkGrayBg).Bold(true).Padding(0, 1).Render("📁 LOCAL OFFLINE VIDEOS")
	} else {
		modeBadge = lipgloss.NewStyle().Background(purpleColor).Foreground(darkGrayBg).Bold(true).Padding(0, 1).Render("🌐 ONLINE YOUTUBE")
	}

	qualityPreset := player.QualityPresets[m.selectedQualityIndex]
	var qColor lipgloss.Color
	switch qualityPreset.ID {
	case "best", "1080":
		qColor = cyanColor
	case "720", "480":
		qColor = yellowColor
	case "audio":
		qColor = greenColor
	default:
		qColor = grayColor
	}
	qualityBadge := lipgloss.NewStyle().Background(qColor).Foreground(darkGrayBg).Bold(true).Padding(0, 1).Render("⚡ QUALITY: " + qualityPreset.Label + " [v]")

	s.WriteString(lipgloss.NewStyle().Align(lipgloss.Center).Width(m.width).Render(fmt.Sprintf("%s   %s", modeBadge, qualityBadge)))
	s.WriteString("\n\n")

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
		resultsContent = "\n" + lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(infoStyle.Render("⚡ Scraping YouTube video metadata asynchronously...\n\nPlease wait a moment.")) + "\n"
	} else if m.searchErr != nil {
		resultsContent = "\n" + lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(errorStyle.Render(fmt.Sprintf("⚠️ Search Failed:\n\n%v", m.searchErr))) + "\n"
	} else if len(m.results) == 0 {
		if m.isLocalMode {
			resultsContent = "\n" + lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(infoStyle.Render("No local video files found in current directory or ./downloads.\n\nPress [m] to switch to Online YouTube mode.")) + "\n"
		} else {
			resultsContent = "\n" + lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(infoStyle.Render("No search results found.\n\nPress [/] to search YouTube, or [m] for Local Offline Video mode.")) + "\n"
		}
	} else {
		var sb strings.Builder
		headerStr := fmt.Sprintf("  %-*s  %-*s  %6s", maxTitleLen, "TITLE", maxChanLen, "UPLOADER", "DURATION")
		headerStyled := lipgloss.NewStyle().Foreground(purpleColor).Bold(true).Underline(true).Render(headerStr)
		sb.WriteString(headerStyled + "\n\n")

		pageSize := mainContentHeight
		if m.cursor < m.startIdx {
			m.startIdx = m.cursor
		} else if m.cursor >= m.startIdx+pageSize {
			m.startIdx = m.cursor - pageSize + 1
		}

		if m.startIdx < 0 {
			m.startIdx = 0
		}
		if m.startIdx > len(m.results)-pageSize {
			m.startIdx = len(m.results) - pageSize
		}
		if m.startIdx < 0 {
			m.startIdx = 0
		}

		for i := m.startIdx; i < m.startIdx+pageSize && i < len(m.results); i++ {
			video := m.results[i]
			title := truncate(video.Title, maxTitleLen)
			uploader := truncate(video.Uploader, maxChanLen)
			duration := video.FormatDuration()

			var rowStr string
			if i == m.cursor {
				// Highlight selected row
				rawRow := fmt.Sprintf("▶ %-*s  %-*s  %6s", maxTitleLen, title, maxChanLen, uploader, duration)
				rawRow = fmt.Sprintf("%-*s", innerWidth-2, rawRow)
				rowStr = selectedRowStyle.Render(rawRow)
			} else {
				// Regular row
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

	// Bottom Panel Section
	var playerContent strings.Builder
	
	if m.downloading {
		badge := lipgloss.NewStyle().Background(pinkColor).Foreground(darkGrayBg).Bold(true).Padding(0, 1).Render("📥 DOWNLOADING")
		var currentTitle string
		for _, v := range m.results {
			if v.ID == m.downloadVideoID {
				currentTitle = v.Title
				break
			}
		}
		titleStyled := lipgloss.NewStyle().Foreground(pinkColor).Bold(true).Render(truncate(currentTitle, m.width-25))
		playerContent.WriteString(fmt.Sprintf("%s  %s\n\n", badge, titleStyled))

		progBarWidth := m.width - 32
		if progBarWidth < 10 {
			progBarWidth = 10
		}
		
		timelineStr := fmt.Sprintf("Progress: [%s] %5.1f%%  │  Merging Streams (%s)...",
			renderCustomProgressBar(progBarWidth, m.downloadPercent/100.0, pinkColor),
			m.downloadPercent,
			qualityPreset.Label,
		)
		playerContent.WriteString(timelineStr)
	} else if m.downloadErr != nil {
		badge := lipgloss.NewStyle().Background(lipgloss.Color("#ff007f")).Foreground(darkGrayBg).Bold(true).Padding(0, 1).Render("⚠️ ERROR")
		errStyled := errorStyle.Render(fmt.Sprintf("Download failed: %v", m.downloadErr))
		playerContent.WriteString(fmt.Sprintf("%s  %s\n\n", badge, errStyled))
		playerContent.WriteString(lipgloss.NewStyle().Foreground(grayColor).Render("Press Enter to stream, or 'd' to download again."))
	} else if m.playbackErr != nil {
		badge := lipgloss.NewStyle().Background(lipgloss.Color("#ff007f")).Foreground(darkGrayBg).Bold(true).Padding(0, 1).Render("⚠️ MPV ERROR")
		errStyled := errorStyle.Render(fmt.Sprintf("Playback failed: %v", m.playbackErr))
		playerContent.WriteString(fmt.Sprintf("%s  %s\n\n", badge, errStyled))
		playerContent.WriteString(lipgloss.NewStyle().Foreground(grayColor).Render("Ensure mpv is installed correctly or try changing video driver."))
	} else if m.downloadSuccessMsg != "" {
		badge := lipgloss.NewStyle().Background(greenColor).Foreground(darkGrayBg).Bold(true).Padding(0, 1).Render("💾 SAVED")
		titleStyled := lipgloss.NewStyle().Foreground(greenColor).Bold(true).Render(m.downloadSuccessMsg)
		playerContent.WriteString(fmt.Sprintf("%s  %s\n\n", badge, titleStyled))
		playerContent.WriteString(lipgloss.NewStyle().Foreground(grayColor).Render("Press Enter to play the downloaded local file instantly via mpv."))
	} else {
		badge := lipgloss.NewStyle().Background(grayColor).Foreground(darkGrayBg).Bold(true).Padding(0, 1).Render("■ READY")
		titleStyled := lipgloss.NewStyle().Foreground(grayColor).Render("Select a video from the list")
		playerContent.WriteString(fmt.Sprintf("%s  %s\n\n", badge, titleStyled))
		qualityInfo := fmt.Sprintf("Quality: %s (%s)  │  [v/Tab] Cycle  [1-6] Quick Select: 1:4K 2:1080 3:720 4:480 5:360 6:Audio",
			greenStatusStyle.Render(qualityPreset.Label),
			qualityPreset.Description,
		)
		playerContent.WriteString(qualityInfo + "\n")
		playerContent.WriteString(lipgloss.NewStyle().Foreground(grayColor).Render("Press Enter to Stream. Press 'd' to download. Press [v] or [1-6] to change quality."))
	}

	s.WriteString(playerBoxStyle.Width(m.width - 4).Render(playerContent.String()))
	s.WriteString("\n")

	// Help Guide / Footer
	keyStyle := lipgloss.NewStyle().Foreground(cyanColor).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(grayColor)
	
	helpStr := fmt.Sprintf(
		" %s %s   %s %s   %s %s   %s %s   %s %s   %s %s   %s %s",
		keyStyle.Render("[/]"), descStyle.Render("Search"),
		keyStyle.Render("[m]"), descStyle.Render("Mode"),
		keyStyle.Render("[v/1-6]"), descStyle.Render("Quality"),
		keyStyle.Render("[Enter]"), descStyle.Render("Play/Stream"),
		keyStyle.Render("[d]"), descStyle.Render("Download"),
		keyStyle.Render("[Esc]"), descStyle.Render("Blur"),
		keyStyle.Render("[q]"), descStyle.Render("Quit"),
	)
	s.WriteString(lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(helpStr))

	return s.String()
}
