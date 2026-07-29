#!/usr/bin/env bash

# ==============================================================================
#  ytmusic / ytsong Installation Script
#  Repository: https://github.com/Ping-Phantom39/yt-cli
# ==============================================================================

set -e
set -o pipefail

# Repo configuration
REPO="Ping-Phantom39/yt-cli"
BINARY_NAME="ytmusic"
SYMLINK_NAME="ytsong"
BINARY_REPO_PATH="yt-song/ytmusic"

# Colors for Cyberpunk aesthetic
COLOR_RESET="\033[0m"
COLOR_CYAN="\033[38;2;0;240;255m\033[1m"
COLOR_PINK="\033[38;2;255;0;127m\033[1m"
COLOR_PURPLE="\033[38;2;121;40;202m\033[1m"
COLOR_GREEN="\033[38;2;57;255;20m\033[1m"
COLOR_YELLOW="\033[38;2;255;255;0m\033[1m"
COLOR_GRAY="\033[38;2;120;120;120m"
COLOR_BOLD="\033[1m"

log_info() {
    echo -e "${COLOR_CYAN}[INFO]${COLOR_RESET} $1"
}

log_success() {
    echo -e "${COLOR_GREEN}[SUCCESS]${COLOR_RESET} $1"
}

log_warn() {
    echo -e "${COLOR_YELLOW}[WARNING]${COLOR_RESET} $1"
}

log_error() {
    echo -e "${COLOR_PINK}[ERROR]${COLOR_RESET} $1" >&2
}

banner() {
    echo -e "${COLOR_CYAN}"
    echo "  ██╗   ██╗████████╗███╗   ███╗██╗   ██╗███████╗██╗ ██████╗"
    echo "  ╚██╗ ██╔╝╚══██╔══╝████╗ ████║██║   ██║██╔════╝██║██╔════╝"
    echo "   ╚████╔╝    ██║   ██╔████╔██║██║   ██║███████╗██║██║     "
    echo "    ╚██╔╝     ██║   ██║╚██╔╝██║██║   ██║╚════██║██║██║     "
    echo "     ██║      ██║   ██║ ╚═╝ ██║╚██████╔╝███████║██║╚██████╗"
    echo "     ╚═╝      ╚═╝   ╚═╝     ╚═╝ ╚═════╝ ╚══════╝╚═╝ ╚═════╝"
    echo -e "${COLOR_RESET}"
    echo -e "${COLOR_PURPLE}  ⚡ Cyberpunk Terminal YouTube Audio Player & Downloader ⚡${COLOR_RESET}"
    echo -e "${COLOR_GRAY}------------------------------------------------------------${COLOR_RESET}"
}

# Check for required tools
need_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        log_error "Required command '$1' is not installed. Please install it and try again."
        exit 1
    fi
}

need_cmd uname
need_cmd chmod
need_cmd mkdir

# HTTP downloader detection (curl or wget)
if command -v curl >/dev/null 2>&1; then
    HTTP_GET="curl -fsSL"
    HTTP_GET_HEADER="curl -fsSL -H"
elif command -v wget >/dev/null 2>&1; then
    HTTP_GET="wget -qO-"
    HTTP_GET_HEADER="wget -qO- --header"
else
    log_error "Neither 'curl' nor 'wget' was found. Please install one of them to proceed."
    exit 1
fi

# Detect Operating System
detect_os() {
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        linux*)   echo "linux" ;;
        darwin*)  echo "darwin" ;;
        freebsd*) echo "freebsd" ;;
        msys*|mingw*|cygwin*) echo "windows" ;;
        *)
            log_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac
}

# Detect Architecture
detect_arch() {
    local arch
    arch="$(uname -m | tr '[:upper:]' '[:lower:]')"
    case "$arch" in
        x86_64|amd64)         echo "amd64" ;;
        i386|i686|x86)        echo "386" ;;
        aarch64|arm64)        echo "arm64" ;;
        armv7l|armv6l|arm)    echo "arm" ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Determine installation directory
determine_install_dir() {
    if [ -n "$BINDIR" ]; then
        echo "$BINDIR"
    elif [ -n "$INSTALL_DIR" ]; then
        echo "$INSTALL_DIR"
    elif [ "$(id -u)" -eq 0 ]; then
        echo "/usr/local/bin"
    elif [ -d "$HOME/.local/bin" ] || mkdir -p "$HOME/.local/bin" 2>/dev/null; then
        echo "$HOME/.local/bin"
    elif [ -d "$HOME/bin" ] || mkdir -p "$HOME/bin" 2>/dev/null; then
        echo "$HOME/bin"
    else
        echo "/usr/local/bin"
    fi
}

# Main Installation Logic
main() {
    banner

    OS="$(detect_os)"
    ARCH="$(detect_arch)"
    log_info "Detected OS: ${COLOR_BOLD}${OS}${COLOR_RESET}, Architecture: ${COLOR_BOLD}${ARCH}${COLOR_RESET}"

    # Setup temporary directory for download
    TMP_DIR=$(mktemp -d -t ytmusic-install-XXXXXX)
    trap 'rm -rf "$TMP_DIR"' EXIT

    DOWNLOADED=0
    FOUND_BINARY="${TMP_DIR}/${BINARY_NAME}"
    TAG="${VERSION:-main}"
    BRANCH="${BRANCH:-main}"

    # Candidate URLs for direct raw binary in repo
    RAW_URLS=(
        "https://raw.githubusercontent.com/${REPO}/${BRANCH}/${BINARY_REPO_PATH}"
        "https://github.com/${REPO}/raw/${BRANCH}/${BINARY_REPO_PATH}"
        "https://github.com/${REPO}/blob/${BRANCH}/${BINARY_REPO_PATH}?raw=true"
    )

    if [ -z "$VERSION" ]; then
        log_info "Attempting to download binary directly from repository (${REPO} @ ${BRANCH})..."
        for url in "${RAW_URLS[@]}"; do
            log_info "Checking asset URL: ${url}..."
            if $HTTP_GET "$url" > "$FOUND_BINARY" 2>/dev/null && [ -s "$FOUND_BINARY" ]; then
                # Ensure downloaded output is not an HTML 404/error page
                if ! head -n 5 "$FOUND_BINARY" | grep -qi "<html"; then
                    log_success "Successfully downloaded binary from repository!"
                    DOWNLOADED=1
                    break
                fi
            fi
            rm -f "$FOUND_BINARY"
        done
    fi

    # Fallback to GitHub Releases if direct repo binary download did not succeed
    if [ "$DOWNLOADED" -eq 0 ]; then
        TARGET_VERSION="${VERSION:-latest}"
        log_info "Fetching release tag / metadata from GitHub (${REPO})..."
        RELEASE_JSON_URL="https://api.github.com/repos/${REPO}/releases/latest"
        
        # Try fetching from GitHub API
        RELEASE_DATA=$($HTTP_GET "$RELEASE_JSON_URL" 2>/dev/null || true)
        
        if [ -n "$RELEASE_DATA" ]; then
            TAG=$(echo "$RELEASE_DATA" | grep -o '"tag_name": *"[^"]*"' | head -n 1 | cut -d '"' -f 4)
        fi

        # Fallback to GitHub release redirect if API call failed or rate limited
        if [ -z "$TAG" ]; then
            log_warn "GitHub API rate limit reached or unavailable. Falling back to release redirect check..."
            REDIRECT_URL=$(curl -fssIL -o /dev/null -w "%{url_effective}" "https://github.com/${REPO}/releases/latest" 2>/dev/null || true)
            TAG="${REDIRECT_URL##*/}"
        fi

        if [ -n "$TAG" ] && [ "$TAG" != "latest" ]; then
            log_info "Selected Release Version: ${COLOR_BOLD}${TAG}${COLOR_RESET}"

            VERSION_NUM="${TAG#v}"

            OS_CAP="$(tr '[:lower:]' '[:upper:]' <<< "${OS:0:1}")${OS:1}"
            if [ "$OS" = "darwin" ]; then OS_CAP="Darwin"; fi
            
            ARCH_ALT="$ARCH"
            if [ "$ARCH" = "amd64" ]; then ARCH_ALT="x86_64"; fi
            if [ "$ARCH" = "arm64" ]; then ARCH_ALT="aarch64"; fi

            DOWNLOAD_URL=""
            ASSET_FILENAME=""

            if [ -n "$RELEASE_DATA" ]; then
                MATCHED_URL=$(echo "$RELEASE_DATA" | grep -o '"browser_download_url": *"[^"]*"' | cut -d '"' -f 4 | grep -i "$OS" | grep -i -E "($ARCH|$ARCH_ALT)" | head -n 1 || true)
                if [ -n "$MATCHED_URL" ]; then
                    DOWNLOAD_URL="$MATCHED_URL"
                    ASSET_FILENAME="$(basename "$DOWNLOAD_URL")"
                fi
            fi

            if [ -z "$DOWNLOAD_URL" ]; then
                CANDIDATES=(
                    "${BINARY_NAME}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
                    "${BINARY_NAME}_${VERSION_NUM}_${OS_CAP}_${ARCH_ALT}.tar.gz"
                    "${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
                    "${BINARY_NAME}-${OS}-${ARCH}.tar.gz"
                    "${BINARY_NAME}_${OS}_${ARCH}"
                    "${BINARY_NAME}-${OS}-${ARCH}"
                    "${BINARY_NAME}_${VERSION_NUM}_${OS}_${ARCH}.zip"
                    "${BINARY_NAME}"
                )

                BASE_RELEASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

                for candidate in "${CANDIDATES[@]}"; do
                    TEST_URL="${BASE_RELEASE_URL}/${candidate}"
                    log_info "Checking asset URL: ${candidate}..."
                    if curl --output /dev/null --silent --head --fail "$TEST_URL"; then
                        DOWNLOAD_URL="$TEST_URL"
                        ASSET_FILENAME="$candidate"
                        break
                    fi
                done
            fi

            if [ -n "$DOWNLOAD_URL" ]; then
                log_info "Downloading asset from: ${COLOR_CYAN}${DOWNLOAD_URL}${COLOR_RESET}"
                ARCHIVE_PATH="${TMP_DIR}/${ASSET_FILENAME}"
                if $HTTP_GET "$DOWNLOAD_URL" > "$ARCHIVE_PATH"; then
                    EXTRACT_DIR="${TMP_DIR}/extracted"
                    mkdir -p "$EXTRACT_DIR"
                    case "$ASSET_FILENAME" in
                        *.tar.gz|*.tgz)
                            need_cmd tar
                            tar -xzf "$ARCHIVE_PATH" -C "$EXTRACT_DIR"
                            ;;
                        *.zip)
                            need_cmd unzip
                            unzip -q "$ARCHIVE_PATH" -d "$EXTRACT_DIR"
                            ;;
                        *)
                            cp "$ARCHIVE_PATH" "${EXTRACT_DIR}/${BINARY_NAME}"
                            ;;
                    esac
                    FOUND_REL_BIN=$(find "$EXTRACT_DIR" -type f \( -name "$BINARY_NAME" -o -name "$SYMLINK_NAME" \) | head -n 1)
                    if [ -n "$FOUND_REL_BIN" ]; then
                        cp "$FOUND_REL_BIN" "$FOUND_BINARY"
                        DOWNLOADED=1
                    fi
                fi
            fi
        fi
    fi

    if [ "$DOWNLOADED" -eq 0 ] || [ ! -f "$FOUND_BINARY" ]; then
        log_error "Could not download ${BINARY_NAME} binary from repository (https://github.com/${REPO}/tree/${BRANCH}/${BINARY_REPO_PATH}) or release assets."
        log_error "Please check the repository URL or specify a valid version/branch."
        exit 1
    fi

    chmod +x "$FOUND_BINARY"

    INSTALL_DIR="$(determine_install_dir)"
    log_info "Target installation directory: ${COLOR_BOLD}${INSTALL_DIR}${COLOR_RESET}"

    # Ensure destination directory exists
    if [ ! -d "$INSTALL_DIR" ]; then
        log_info "Creating directory ${INSTALL_DIR}..."
        mkdir -p "$INSTALL_DIR" 2>/dev/null || sudo mkdir -p "$INSTALL_DIR"
    fi

    TARGET_PATH="${INSTALL_DIR}/${BINARY_NAME}"
    SYMLINK_PATH="${INSTALL_DIR}/${SYMLINK_NAME}"

    log_info "Installing ${BINARY_NAME} to ${TARGET_PATH}..."

    # Perform copy with sudo if permission denied
    if touch "${INSTALL_DIR}/.test_write" 2>/dev/null; then
        rm -f "${INSTALL_DIR}/.test_write"
        cp -f "$FOUND_BINARY" "$TARGET_PATH"
        chmod +x "$TARGET_PATH"
        ln -sf "$TARGET_PATH" "$SYMLINK_PATH" 2>/dev/null || true
    else
        log_warn "Sudo privileges required to install to ${INSTALL_DIR}"
        sudo cp -f "$FOUND_BINARY" "$TARGET_PATH"
        sudo chmod +x "$TARGET_PATH"
        sudo ln -sf "$TARGET_PATH" "$SYMLINK_PATH" 2>/dev/null || true
    fi

    log_success "Successfully installed ${BINARY_NAME} (${TAG}) to ${TARGET_PATH}!"

    # Create alias/symlink message
    if [ -L "$SYMLINK_PATH" ] || [ -f "$SYMLINK_PATH" ]; then
        log_info "Symlink created: ${COLOR_CYAN}${SYMLINK_NAME} -> ${BINARY_NAME}${COLOR_RESET}"
    fi

    # Check if INSTALL_DIR is in PATH
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
        log_warn "${INSTALL_DIR} is not currently in your PATH environment variable."
        log_warn "Add it to your shell configuration file (~/.bashrc, ~/.zshrc, or ~/.profile):"
        echo -e "${COLOR_CYAN}    export PATH=\"${INSTALL_DIR}:\$PATH\"${COLOR_RESET}"
    fi

    echo ""
    log_info "Run ${COLOR_CYAN}${BINARY_NAME} --check${COLOR_RESET} to verify system dependencies (ffmpeg, yt-dlp, etc.)."
    log_success "Setup complete! Enjoy high-fidelity cyberpunk terminal audio playback. 🚀"
}

main "$@"
