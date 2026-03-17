#!/bin/bash

set -e

# AfterMail Build Script
# Supports: build, build-gui, build-cli, build-daemon, clean, install

VERSION="${VERSION:-dev}"
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS="-X main.version=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.gitCommit=${GIT_COMMIT}"

# Default build directory
BUILD_DIR="./bin"
INSTALL_PREFIX="${INSTALL_PREFIX:-/usr/local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Ensure build directory exists
ensure_build_dir() {
    mkdir -p "${BUILD_DIR}"
}

# Build GUI version (full Fyne GUI)
build_gui() {
    info "Building AfterMail GUI..."
    ensure_build_dir
    go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/aftermail" .
    success "Built ${BUILD_DIR}/aftermail (GUI version)"
}

# Build CLI-only version (no GUI, just commands)
build_cli() {
    info "Building AfterMail CLI (headless)..."
    ensure_build_dir
    # Build with GUI disabled by using build tags or a separate main
    # For now, this builds the same binary but could be optimized
    go build -ldflags "${LDFLAGS}" -tags nogui -o "${BUILD_DIR}/aftermail-cli" .
    success "Built ${BUILD_DIR}/aftermail-cli (CLI version)"
}

# Build daemon
build_daemon() {
    info "Building AfterMail daemon..."
    ensure_build_dir
    go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/aftermaild" ./cmd/aftermaild
    success "Built ${BUILD_DIR}/aftermaild (daemon)"
}

# Build all targets
build_all() {
    info "Building all AfterMail components..."
    build_gui
    build_daemon
    success "All components built successfully!"
}

# Clean build artifacts
clean() {
    info "Cleaning build artifacts..."
    rm -rf "${BUILD_DIR}"
    rm -f aftermail aftermaild aftermail-cli
    rm -f *.db
    go clean -cache
    success "Build artifacts cleaned"
}

# Install binaries to system
install() {
    if [ ! -f "${BUILD_DIR}/aftermail" ]; then
        error "No binaries found. Run 'build' first."
        exit 1
    fi

    info "Installing AfterMail to ${INSTALL_PREFIX}..."

    # Check if we need sudo
    if [ -w "${INSTALL_PREFIX}" ]; then
        cp "${BUILD_DIR}/aftermail" "${INSTALL_PREFIX}/aftermail"
        [ -f "${BUILD_DIR}/aftermaild" ] && cp "${BUILD_DIR}/aftermaild" "${INSTALL_PREFIX}/aftermaild"
        success "Installed to ${INSTALL_PREFIX}"
    else
        warn "Need sudo to install to ${INSTALL_PREFIX}"
        sudo cp "${BUILD_DIR}/aftermail" "${INSTALL_PREFIX}/aftermail"
        [ -f "${BUILD_DIR}/aftermaild" ] && sudo cp "${BUILD_DIR}/aftermaild" "${INSTALL_PREFIX}/aftermaild"
        success "Installed to ${INSTALL_PREFIX} (with sudo)"
    fi
}

# Uninstall binaries from system
uninstall() {
    info "Uninstalling AfterMail from ${INSTALL_PREFIX}..."

    if [ -w "${INSTALL_PREFIX}" ]; then
        rm -f "${INSTALL_PREFIX}/aftermail"
        rm -f "${INSTALL_PREFIX}/aftermaild"
        success "Uninstalled from ${INSTALL_PREFIX}"
    else
        warn "Need sudo to uninstall from ${INSTALL_PREFIX}"
        sudo rm -f "${INSTALL_PREFIX}/aftermail"
        sudo rm -f "${INSTALL_PREFIX}/aftermaild"
        success "Uninstalled from ${INSTALL_PREFIX} (with sudo)"
    fi
}

# Run tests
test() {
    info "Running tests..."
    go test -v ./...
    success "All tests passed"
}

# Generate protobuf files
generate_proto() {
    info "Regenerating protobuf files..."
    export PATH=$PATH:$(go env GOPATH)/bin
    cd pkg/proto
    protoc --go_out=. --go_opt=paths=source_relative \
           --go-grpc_out=. --go-grpc_opt=paths=source_relative \
           *.proto
    cd ../..
    success "Protobuf files regenerated"
}

# Development build (faster, no optimizations)
dev() {
    info "Building AfterMail (dev mode, no optimizations)..."
    ensure_build_dir
    go build -o "${BUILD_DIR}/aftermail" .
    go build -o "${BUILD_DIR}/aftermaild" ./cmd/aftermaild
    success "Dev build complete"
}

# Debug build (with symbols for dlv)
debug() {
    info "Building AfterMail (debug mode with DWARF symbols)..."
    ensure_build_dir
    go build -gcflags="all=-N -l" -o "${BUILD_DIR}/aftermail" .
    go build -gcflags="all=-N -l" -o "${BUILD_DIR}/aftermaild" ./cmd/aftermaild
    success "Debug build complete"
}

# Release build (with optimizations)
release() {
    info "Building AfterMail (release mode)..."
    ensure_build_dir

    # Build for current platform with optimizations
    CGO_ENABLED=1 go build -ldflags "${LDFLAGS} -s -w" -trimpath -o "${BUILD_DIR}/aftermail" .
    CGO_ENABLED=1 go build -ldflags "${LDFLAGS} -s -w" -trimpath -o "${BUILD_DIR}/aftermaild" ./cmd/aftermaild

    success "Release build complete"
}

# Cross-compile for multiple platforms
cross_compile() {
    info "Cross-compiling for multiple platforms..."
    ensure_build_dir

    platforms=("darwin/amd64" "darwin/arm64" "linux/amd64" "linux/arm64" "windows/amd64")

    for platform in "${platforms[@]}"; do
        platform_split=(${platform//\// })
        GOOS=${platform_split[0]}
        GOARCH=${platform_split[1]}
        output_name="${BUILD_DIR}/aftermail-${GOOS}-${GOARCH}"
        daemon_output="${BUILD_DIR}/aftermaild-${GOOS}-${GOARCH}"

        if [ "$GOOS" = "windows" ]; then
            output_name+='.exe'
            daemon_output+='.exe'
        fi

        info "Building for ${GOOS}/${GOARCH}..."

        # GUI build (may fail on some platforms without CGO)
        if GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=1 go build -ldflags "${LDFLAGS}" -o "${output_name}" . 2>/dev/null; then
            success "Built ${output_name}"
        else
            warn "Skipped GUI for ${GOOS}/${GOARCH} (CGO required)"
        fi

        # Daemon build (always works)
        GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "${LDFLAGS}" -o "${daemon_output}" ./cmd/aftermaild
        success "Built ${daemon_output}"
    done

    success "Cross-compilation complete"
}

# Show usage
usage() {
    cat <<EOF
AfterMail Build Script

Usage: $0 <command> [options]

Commands:
    build|all       Build all components (GUI + daemon)
    gui-only        Build GUI application only
    daemon-only     Build daemon only
    debug           Build with debug symbols (for dlv)
    clean           Remove build artifacts
    install         Install binaries to system (default: /usr/local/bin)
    
    # Legacy / Additional targets
    build-gui       Build GUI application only
    build-cli       Build CLI-only version (headless)
    build-daemon    Build daemon only
    dev             Quick development build (no optimizations)
    release         Optimized release build
    cross           Cross-compile for multiple platforms

    clean           Remove build artifacts
    install         Install binaries to system (default: /usr/local/bin)
    uninstall       Remove binaries from system

    test            Run tests
    proto           Regenerate protobuf files

    help            Show this help message

Environment Variables:
    VERSION         Version string (default: dev)
    INSTALL_PREFIX  Installation directory (default: /usr/local/bin)

Examples:
    $0 build                    # Build everything
    $0 clean build              # Clean then build
    $0 release install          # Release build and install
    VERSION=1.0.0 $0 release    # Build release with version

EOF
}

# Main command dispatcher
main() {
    if [ $# -eq 0 ]; then
        usage
        exit 0
    fi

    while [ $# -gt 0 ]; do
        case "$1" in
            build|all)
                build_all
                ;;
            build-gui|gui-only)
                build_gui
                ;;
            build-cli)
                build_cli
                ;;
            build-daemon|daemon-only)
                build_daemon
                ;;
            dev)
                dev
                ;;
            debug)
                debug
                ;;
            release)
                release
                ;;
            cross)
                cross_compile
                ;;
            clean)
                clean
                ;;
            install)
                install
                ;;
            uninstall)
                uninstall
                ;;
            test)
                test
                ;;
            proto)
                generate_proto
                ;;
            help|--help|-h)
                usage
                exit 0
                ;;
            *)
                error "Unknown command: $1"
                usage
                exit 1
                ;;
        esac
        shift
    done
}

# Run main
main "$@"
