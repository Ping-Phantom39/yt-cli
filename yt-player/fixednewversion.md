# Bug Report & Fix Documentation: YouTube Stream 429 & Video Quality Selection

## 1. Problem Overview

### A. Initial Issue: 429 Too Many Requests / Format Unavailable
When running `ytplayer` to stream YouTube videos, playback failed with the following error output:

```text
[ytdl_hook] ERROR: [youtube] lAbWm-DIB-E: Requested format is not available. Use --list-formats for a list of available formats
[ytdl_hook] youtube-dl failed: unexpected error occurred
[ffmpeg] https: HTTP error 429 Too Many Requests
Failed to open https://www.youtube.com/watch?v=lAbWm-DIB-E.
Exiting... (Errors when loading file)
```

### B. Secondary Issue: Low Quality (360p) Playback
When defaulting solely to the `android` player client, YouTube only returned legacy single-stream formats (format 18 = 360p H.264/AAC), resulting in low-resolution video playback.

---

## 2. Root Cause Analysis

1. **YouTube SABR & PO Token Enforcement**: YouTube restricts default `web` and `mweb` client requests, requiring GVS Proof-of-Origin (PO) tokens and SABR decoders. Requests without them return HTTP 429/403 or only storyboard images.
2. **Android Client Format Limitations**: While the basic `android` client bypasses bot challenges, it only serves format 18 (360p).
3. **High-Definition Stream Unlocking via `visionos` Client**: The `visionos` (Apple Vision Pro) client delivers Full HD (1080p), 1440p, 4K (2160p), and high-fidelity Opus (48kHz) / AAC (128kbps) streams without requiring PO tokens or triggering 429/403 blocks.
4. **Missing MPV Format Selectors**: `mpv` required `--ytdl-format="bestvideo+bestaudio/best"` and proper `--ytdl-raw-options` to request the highest resolution video track merged with the highest quality audio track.

---

## 3. Implemented Solutions & Interactive Quality Selection

### 1. Interactive Video Quality Selector (TUI)
Users can change playback quality live in the interface:
- **`[v]` or `[Tab]`**: Cycles through video quality presets in real-time.
- **`[1] - [6]`**: Quick jump to a specific preset:
  - `[1]`: **Best / 4K** (Max resolution available - 4K/1440p/1080p)
  - `[2]`: **1080p FHD** (Full High Definition)
  - `[3]`: **720p HD** (High Definition)
  - `[4]`: **480p SD** (Standard Definition)
  - `[5]`: **360p Saver** (Data Saver)
  - `[6]`: **Audio Only 🎵** (Music & Podcasts - zero video bandwidth)
- **Visual Cyberpunk Badge**: Displays active quality in the header (e.g. `[ ⚡ QUALITY: 1080p FHD [v] ]`).
- **Player Status Panel**: Shows currently selected quality, resolution description, and shortcuts.

### 2. High-Definition Streaming in `mpv` (`internal/player/player.go`)
- **`visionos` Client**: Configured `--ytdl-raw-options="extractor-args=youtube:player_client=visionos"` to unlock HD and 4K streams.
- **Dynamic Quality Formats**:
  - `audio`: `--ytdl-format=bestaudio/best --no-video`
  - `1080`: `--ytdl-format=bestvideo[height<=1080]+bestaudio/best[height<=1080]/best`
  - `720`: `--ytdl-format=bestvideo[height<=720]+bestaudio/best[height<=720]/best`
  - `480`: `--ytdl-format=bestvideo[height<=480]+bestaudio/best[height<=480]/best`
  - `360`: `--ytdl-format=bestvideo[height<=360]+bestaudio/best[height<=360]/best/18`
  - `best`: `--ytdl-format=bestvideo+bestaudio/best`

### 3. CLI Quality Flag (`cmd/root.go`)
- Added `-q, --quality` flag allowing users to launch the player with a custom default quality:
  ```bash
  ytplayer "query"                # Plays at the best quality available (1080p / 4K)
  ytplayer -q 1080 "query"        # Max 1080p Full HD
  ytplayer -q 720 "query"         # Max 720p HD
  ytplayer -q audio "query"       # Audio only mode (music player)
  ```

### 4. Quality-Aware Downloader (`internal/downloader/downloader.go`)
- Updated `Download()` to download video/audio matching the user's selected quality preset.
- Utilizes `visionos,android_creator,tv_embedded,android,ios,tv,web` clients for multi-resolution downloads.

---

## 4. Modified Files Summary

| File | Summary of Changes |
| :--- | :--- |
| [`internal/player/player.go`](file:///home/kamal/Documents/YT_SONGS/yt-cli/yt-player/internal/player/player.go) | Configured `QualityPresets`, `QualityOption`, format selectors, and `visionos` options. |
| [`internal/ui/ui.go`](file:///home/kamal/Documents/YT_SONGS/yt-cli/yt-player/internal/ui/ui.go) | Added interactive quality cycling (`[v]`/`[Tab]`), direct numeric shortcuts (`[1-6]`), and quality badges. |
| [`cmd/root.go`](file:///home/kamal/Documents/YT_SONGS/yt-cli/yt-player/cmd/root.go) | Added `-q, --quality` flag for user-selectable initial video resolution. |
| [`internal/downloader/downloader.go`](file:///home/kamal/Documents/YT_SONGS/yt-cli/yt-player/internal/downloader/downloader.go) | Enabled quality-aware downloads and multi-client resolution extraction. |
| [`internal/player/player_test.go`](file:///home/kamal/Documents/YT_SONGS/yt-cli/yt-player/internal/player/player_test.go) | Added unit tests covering all quality options (Best, 720p, Audio Only). |
| [`README.md`](file:///home/kamal/Documents/YT_SONGS/yt-cli/yt-player/README.md) | Documented quality selector keybindings and CLI flag. |

---

## 5. Verification Results

1. **Unit Tests**:
   ```bash
   go test ./...
   # ok ytplayer/internal/downloader
   # ok ytplayer/internal/player
   ```
2. **Stream Resolution Verification**:
   - `4K / Best`: Verified streaming in **3840x2160 AV1 / 1920x1080 VP9** + **Opus 48kHz** (Exit Code 0).
   - `720p HD`: Verified streaming in **1280x720 AV1/VP9** (Exit Code 0).
   - `Audio Only`: Verified streaming with `--no-video` (Exit Code 0).
