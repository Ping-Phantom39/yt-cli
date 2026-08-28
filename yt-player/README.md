# ytplayer ⚡ Cyberpunk Terminal YouTube Video Player

A high-performance cyberpunk terminal video searcher, downloader, and mpv-backed streaming player. Built in Go, leveraging `yt-dlp`, Bubble Tea, and `mpv`.

`ytplayer` allows you to search YouTube videos directly from your terminal, download them as high-quality MP4s, or stream them instantly using `mpv`. It supports browser cookies or a custom `cookies.txt` file to bypass YouTube's anti-bot blockages.

---

## Features
- **Instant Streaming**: Play videos directly using `mpv` without downloading the full file first.
- **Background Downloader**: Download the best available video + audio tracks and merge them into a single high-quality `.mp4` file saved with the video's title.
- **Local Offline Video Mode**: Play local offline video files (`.mp4`, `.mkv`, `.avi`, `.webm`, `.mov`, etc.) from your current directory and `./downloads/` folder with real-time title filtering.
- **Cyberpunk UI**: Bubble Tea-powered terminal user interface with glowing cyan, pink, and purple aesthetics.
- **Headless Mode Support**: Auto-detects terminal environments (missing GUI/X11 displays) and fallback to terminal-friendly rendering (`--vo=tct` or customized driver).
- **Cookies & Bypass Auth**: Supports passing `--cookies` and `--cookies-from-browser` flags, automatically resolving challenges.
- **Full Viewport Control**: Scroll and query effortlessly.

---

## Installation & Setup

### 1. Requirements
Ensure the following tools are installed on your system:
- **Go 1.25+** (to compile)
- **mpv** (for playback)
- **ffmpeg** (needed by `yt-dlp` to merge stream files)
- **Node.js** or **Deno** (optional, recommended to help bypass YouTube blocks)

To install dependencies on Debian/Ubuntu:
```bash
sudo apt-get update
sudo apt-get install -y mpv ffmpeg nodejs
```

### 2. Compile the Project
Clone or navigate to the directory and run:
```bash
go build -o ytplayer main.go

Alternatively you can download through go installer as:
module github.com/Ping-Phantom39/yt-cli/yt-player
module github.com/Ping-Phantom39/yt-cli/yt-song
```

---

## Usage

Get your  YT Cookies(Netscape format) From Get cookies.txt LOCALLY 
Save it as cookies.txt file 


Start the player in search mode:
```bash
./ytplayer
```

Start directly in Local Offline Video mode:
```bash
./ytplayer --local
```

Start the player and search immediately:
```bash
./ytplayer "lofi hip hop beats"
```

Force terminal rendering (useful for SSH/headless servers):
```bash
./ytplayer --vo tct "gaming lofi"
```

Bypass bot checks with a custom cookies file:
```bash
./ytplayer --cookies cookies.txt "cyberpunk synthwave"
```

Scan and check system dependencies:
```bash
./ytplayer --check
```

### Options and Flags
- `-m, --local`: Start directly in Local Offline Video mode.
- `-l, --limit int`: Number of search results to fetch (default `15`).
- `-q, --quality string`: Max playback quality (e.g. `best`, `1080`, `720`, `480`, `360`) (default `best`).
- `--cookies string`: Path to cookies file (automatically detects `cookies.txt` in current directory if present).
- `--cookies-from-browser string`: Load cookies from a specific browser (e.g., `chrome`, `firefox`, `edge`, `brave`).
- `--vo string`: Force a custom `mpv` video output driver (e.g. `tct`, `sixel`, `kitty`, `gpu`).
- `--check`: Verify that all system dependencies are installed correctly.

---

## Keybindings (TUI)
- `[/]` - Focus the search bar to enter a new query or filter local videos.
- `[m]` - Toggle between **Online YouTube Mode** and **Local Offline Video Mode**.
- `[v]` or `[Tab]` - Cycle through video quality presets (**Best/4K** ➔ **1080p FHD** ➔ **720p HD** ➔ **480p SD** ➔ **360p Saver** ➔ **Audio Only 🎵**).
- `[1] - [6]` - Quick jump to specific quality preset:
  - `[1]`: Best / 4K (Ultra HD)
  - `[2]`: 1080p (Full HD)
  - `[3]`: 720p (High Definition)
  - `[4]`: 480p (Standard Definition)
  - `[5]`: 360p (Data Saver)
  - `[6]`: Audio Only (Music & Podcasts)
- `[Enter]` - Stream and play the selected video instantly using `mpv` with the chosen quality.
- `[d]` - Download the video permanently to `./downloads/` folder with the chosen quality.
- `[Esc]` - Blur/unfocus the search bar.
- `[q]` or `[Ctrl+C]` - Quit the application.

---

## How It Works
1. **Search**: Spawns `yt-dlp` in flat-playlist mode, parsing stdout JSON streams as they arrive.
2. **Playback (Stream/File)**: Uses Bubble Tea's `tea.ExecProcess`. It suspends the TUI, hands standard input/output over to `mpv`, and resumes the TUI seamlessly once `mpv` exits.
3. **Download**: Triggers `yt-dlp` to download high-quality video and audio feeds separately and merges them via `ffmpeg` into a single `.mp4` file named after the video's title. A progress bar updates in real-time in the bottom panel.
4. **Local Offline Mode**: Scans local directories for video files and lets you browse, filter, and play them directly without an internet connection.
