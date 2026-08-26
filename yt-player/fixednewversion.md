# Bug Report & Fix Documentation: YouTube Stream 429 & Format Unavailable Error

## 1. Problem Overview

When running `ytplayer` to stream videos such as `lAbWm-DIB-E` or `fCkeLBGSINs`, playback failed with the following error output:

```text
[ytdl_hook] ERROR: [youtube] lAbWm-DIB-E: Requested format is not available. Use --list-formats for a list of available formats
[ytdl_hook] youtube-dl failed: unexpected error occurred
[ffmpeg] https: HTTP error 429 Too Many Requests
Failed to open https://www.youtube.com/watch?v=lAbWm-DIB-E.
Exiting... (Errors when loading file)
```

---

## 2. Root Cause Analysis

### A. YouTube Bot Verification & SABR Stream Changes
YouTube introduced strict server-side changes targeting the default `web` and `mweb` client requests:
- **Proof-of-Origin (PO) Token Requirement**: Requests coming from web/mweb endpoints without a valid GVS PO token or solver are skipped or rejected with `403 Forbidden` / `429 Too Many Requests`.
- **SABR Streaming Protocol**: Standard web formats are restricted to streaming experiments that return no downloadable stream URLs, leaving only image storyboards (`sb0`, `sb1`, `sb2`, `sb3`).

### B. Missing `ytdl-raw-options` in MPV Hook
In `internal/player/player.go`, when `ytplayer` passed a remote YouTube URL to `mpv`, `mpv`'s internal `ytdl_hook` executed `yt-dlp` with default options (web client):
- Because no extractor client or JavaScript runtime options were forwarded to `mpv`, `yt-dlp` failed format extraction.
- When `yt-dlp` failed, `mpv` attempted to open the raw YouTube webpage URL directly using `ffmpeg`, causing `[ffmpeg] https: HTTP error 429 Too Many Requests`.

### C. Outdated Local `bin/yt-dlp`
The project's local `./bin/yt-dlp` was an older build (`2026.07.04`), whereas YouTube API extraction fixes require the latest `yt-dlp` release (`2026.08.19`+).

---

## 3. Implemented Fixes

### 1. MPV Playback Configuration (`internal/player/player.go`)
- **Android Player Client Routing**: Added `extractor-args=youtube:player_client=android` to `mpv`'s `--ytdl-raw-options`. The Android client delivers direct HTTPS MP4/AAC streams without requiring PO tokens or triggering 429 blocks.
- **Dynamic JS Runtime Detection**: Automatically detects if `node` or `deno` is available on the system and attaches `js-runtimes=node` (or `deno`) into `--ytdl-raw-options`.

### 2. Search & Download Pipeline (`internal/downloader/downloader.go`)
- **Multi-Client Fallback**: Added `--extractor-args "youtube:player_client=android,ios,tv,web"` to both `Search()` and `Download()` to maximize format compatibility.
- **Format Fallback**: Added fallback format `/18` to the format selector string (`bestvideo[ext=mp4]+bestaudio[ext=m4a]/bestvideo+bestaudio/best[ext=mp4]/best/18`).
- **Dynamic JS Runtime**: Added automatic detection for `node`/`deno` during search and download operations.
- **Binary Resolution**: Enhanced `ResolvePath()` and `ResolveFFmpegPath()` to check:
  1. Current working directory (`./bin`)
  2. Directory adjacent to the running executable (`<exe_dir>/yt-dlp` or `<exe_dir>/bin/yt-dlp`)
  3. System `PATH` (`exec.LookPath`)

### 3. Dependency Checker (`internal/deps/deps.go`)
- Refactored `deps.CheckAll()` to use the unified resolution methods (`downloader.ResolvePath()` and `downloader.ResolveFFmpegPath()`).

### 4. Binary Update
- Updated `./bin/yt-dlp` to the latest release (`2026.08.19`).
- Recompiled and installed the new `ytplayer` binary to `~/.local/bin/ytplayer`.

---

## 4. Modified Files Summary

| File | Changes |
| :--- | :--- |
| [`internal/player/player.go`](file:///home/kamal/Documents/YT_SONGS/yt-cli/yt-player/internal/player/player.go) | Configured `--ytdl-raw-options` with Android client and dynamic JS runtimes. |
| [`internal/downloader/downloader.go`](file:///home/kamal/Documents/YT_SONGS/yt-cli/yt-player/internal/downloader/downloader.go) | Added multi-client extractor args, format `/18` fallback, and executable-relative binary path resolution. |
| [`internal/deps/deps.go`](file:///home/kamal/Documents/YT_SONGS/yt-cli/yt-player/internal/deps/deps.go) | Unified dependency path resolution with `downloader`. |
| [`internal/player/player_test.go`](file:///home/kamal/Documents/YT_SONGS/yt-cli/yt-player/internal/player/player_test.go) | Added unit tests verifying generated `mpv` command-line arguments. |
| `bin/yt-dlp` | Upgraded binary to version `2026.08.19`. |

---

## 5. Verification Results

1. **Unit Tests**:
   ```bash
   go test ./...
   # ok ytplayer/internal/downloader
   # ok ytplayer/internal/player
   ```
2. **Dependency Scanner**:
   ```bash
   ytplayer --check
   # [🟢 OK] yt-dlp (v2026.08.19)
   # [🟢 OK] ffmpeg (v7.1.1)
   # [🟢 OK] mpv (v0.40.0)
   # [🟢 OK] Node.js (v20.18.1)
   ```
3. **Playback Verification**:
   - Tested stream extraction on `lAbWm-DIB-E`: Successful (Exit Code 0).
   - Tested stream extraction on `fCkeLBGSINs`: Successful (Exit Code 0).
   - Tested search scraping with query metadata: Successful.
