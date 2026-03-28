#!/bin/sh
set -e

REPO="vaughnbosu/cws-cli"
BINARY="cws"
BASE_URL="https://github.com/${REPO}/releases/latest/download"

# Allow custom install directory via env var
INSTALL_DIR="${CWS_INSTALL_DIR:-}"

main() {
    os=$(detect_os)
    arch=$(detect_arch)
    archive="${BINARY}_${os}_${arch}.tar.gz"

    printf "Installing cws (%s/%s)...\n" "$os" "$arch"

    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    download "$BASE_URL/$archive" "$tmpdir/$archive"
    download "$BASE_URL/checksums.txt" "$tmpdir/checksums.txt"

    verify_checksum "$tmpdir" "$archive" "$os"

    tar xzf "$tmpdir/$archive" -C "$tmpdir"

    install_dir=$(resolve_install_dir)
    install_binary "$tmpdir/$BINARY" "$install_dir"

    printf "Successfully installed %s to %s/%s\n" "$BINARY" "$install_dir" "$BINARY"
    check_path "$install_dir"
}

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       printf "Error: unsupported OS: %s\n" "$(uname -s)" >&2; exit 1 ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             printf "Error: unsupported architecture: %s\n" "$(uname -m)" >&2; exit 1 ;;
    esac
}

download() {
    url="$1"
    dest="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$dest" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$dest" "$url"
    else
        printf "Error: curl or wget is required\n" >&2
        exit 1
    fi
}

verify_checksum() {
    dir="$1"
    file="$2"
    os="$3"

    expected=$(grep "$file" "$dir/checksums.txt" | awk '{print $1}')
    if [ -z "$expected" ]; then
        printf "Error: checksum not found for %s\n" "$file" >&2
        exit 1
    fi

    if [ "$os" = "darwin" ]; then
        actual=$(shasum -a 256 "$dir/$file" | awk '{print $1}')
    else
        actual=$(sha256sum "$dir/$file" | awk '{print $1}')
    fi

    if [ "$expected" != "$actual" ]; then
        printf "Error: checksum verification failed\n" >&2
        printf "  expected: %s\n" "$expected" >&2
        printf "  actual:   %s\n" "$actual" >&2
        exit 1
    fi
}

resolve_install_dir() {
    if [ -n "$INSTALL_DIR" ]; then
        mkdir -p "$INSTALL_DIR"
        echo "$INSTALL_DIR"
        return
    fi

    if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
        echo "/usr/local/bin"
        return
    fi

    fallback="$HOME/.local/bin"
    mkdir -p "$fallback"
    echo "$fallback"
}

install_binary() {
    src="$1"
    dir="$2"

    if [ ! -w "$dir" ]; then
        printf "Error: %s is not writable\n" "$dir" >&2
        printf "Re-run with sudo or set CWS_INSTALL_DIR to a writable directory:\n" >&2
        printf "  sudo sh -c 'curl -fsSL https://vaughnbosu.github.io/cws-cli/install.sh | sh'\n" >&2
        printf "  CWS_INSTALL_DIR=~/.local/bin curl -fsSL https://vaughnbosu.github.io/cws-cli/install.sh | sh\n" >&2
        exit 1
    fi

    install -m 755 "$src" "$dir/$BINARY"
}

check_path() {
    dir="$1"
    case ":$PATH:" in
        *":$dir:"*) ;;
        *)
            printf "\nWarning: %s is not in your PATH\n" "$dir"
            printf "Add it to your shell profile:\n"
            printf "  export PATH=\"%s:\$PATH\"\n" "$dir"
            ;;
    esac
}

main
