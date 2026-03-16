# Meowmail Installation Guide

This guide covers installation and setup for Meowmail on all supported platforms.

## Table of Contents

- [System Requirements](#system-requirements)
- [Quick Start](#quick-start)
- [Installation from Binary](#installation-from-binary)
- [Building from Source](#building-from-source)
- [Configuration](#configuration)
- [First-Time Setup](#first-time-setup)
- [Troubleshooting](#troubleshooting)

## System Requirements

### Minimum Requirements

- **OS**: macOS 11+, Windows 10+, or Linux (Ubuntu 20.04+, Debian 11+, Fedora 35+)
- **RAM**: 2 GB
- **Disk**: 500 MB free space
- **Network**: Internet connection for email sync

### Recommended Requirements

- **OS**: macOS 13+, Windows 11+, or Linux (latest LTS)
- **RAM**: 4 GB
- **Disk**: 2 GB free space (for message cache)
- **Network**: Broadband internet

### Dependencies

Meowmail has no external runtime dependencies - everything is bundled in the binary.

For building from source, you'll need:
- Go 1.21 or later
- protoc (Protocol Buffer compiler)
- Git

## Quick Start

### macOS (Homebrew)

```bash
# Coming soon
brew install meowmail
meowmail
```

### Linux (apt/deb-based)

```bash
# Coming soon
sudo apt install meowmail
meowmail
```

### Windows (Chocolatey)

```powershell
# Coming soon
choco install meowmail
meowmail
```

## Installation from Binary

### macOS

1. **Download the latest release:**
   ```bash
   curl -LO https://github.com/ryan/meowmail/releases/latest/download/meowmail-darwin-amd64
   ```

2. **Make it executable:**
   ```bash
   chmod +x meowmail-darwin-amd64
   ```

3. **Move to your PATH:**
   ```bash
   sudo mv meowmail-darwin-amd64 /usr/local/bin/meowmail
   ```

4. **Allow in Security & Privacy:**
   - Go to System Preferences → Security & Privacy
   - Click "Open Anyway" when prompted about meowmail

5. **Run Meowmail:**
   ```bash
   meowmail
   ```

### Linux

1. **Download the latest release:**
   ```bash
   wget https://github.com/ryan/meowmail/releases/latest/download/meowmail-linux-amd64
   ```

2. **Make it executable:**
   ```bash
   chmod +x meowmail-linux-amd64
   ```

3. **Move to your PATH:**
   ```bash
   sudo mv meowmail-linux-amd64 /usr/local/bin/meowmail
   ```

4. **Install required libraries (if needed):**
   ```bash
   # Ubuntu/Debian
   sudo apt install libgl1-mesa-glx libxi6 libxrandr2 libxcursor1 libxinerama1

   # Fedora/RHEL
   sudo dnf install mesa-libGL libXi libXrandr libXcursor libXinerama
   ```

5. **Run Meowmail:**
   ```bash
   meowmail
   ```

### Windows

1. **Download the latest release:**
   - Visit https://github.com/ryan/meowmail/releases/latest
   - Download `meowmail-windows-amd64.exe`

2. **Move to Program Files:**
   ```powershell
   Move-Item meowmail-windows-amd64.exe "C:\Program Files\Meowmail\meowmail.exe"
   ```

3. **Add to PATH (optional):**
   - Right-click "This PC" → Properties → Advanced System Settings
   - Environment Variables → Path → Edit
   - Add: `C:\Program Files\Meowmail`

4. **Run Meowmail:**
   - Double-click `meowmail.exe` or run from terminal:
   ```powershell
   meowmail
   ```

## Building from Source

### Prerequisites

1. **Install Go:**
   ```bash
   # macOS
   brew install go

   # Linux
   wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
   export PATH=$PATH:/usr/local/go/bin

   # Windows
   # Download and install from https://go.dev/dl/
   ```

2. **Install Protocol Buffer Compiler:**
   ```bash
   # macOS
   brew install protobuf

   # Linux
   sudo apt install protobuf-compiler  # Ubuntu/Debian
   sudo dnf install protobuf-compiler  # Fedora/RHEL

   # Windows
   # Download from https://github.com/protocolbuffers/protobuf/releases
   ```

3. **Install Go protobuf plugins:**
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

   # Add Go bin to PATH
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

### Build Steps

1. **Clone the repository:**
   ```bash
   git clone https://github.com/ryan/meowmail.git
   cd meowmail
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Generate protobuf code (if needed):**
   ```bash
   cd pkg/proto
   protoc --go_out=. --go_opt=paths=source_relative \
          --go-grpc_out=. --go-grpc_opt=paths=source_relative \
          amp.proto client.proto
   cd ../..
   ```

4. **Build:**
   ```bash
   go build -o meowmail
   ```

5. **Install system-wide (optional):**
   ```bash
   # macOS/Linux
   sudo mv meowmail /usr/local/bin/

   # Windows
   Move-Item meowmail.exe "C:\Program Files\Meowmail\"
   ```

### Cross-Platform Builds

Build for all platforms from any OS:

```bash
# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o meowmail-darwin-amd64

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o meowmail-darwin-arm64

# Linux (x64)
GOOS=linux GOARCH=amd64 go build -o meowmail-linux-amd64

# Linux (ARM)
GOOS=linux GOARCH=arm64 go build -o meowmail-linux-arm64

# Windows (x64)
GOOS=windows GOARCH=amd64 go build -o meowmail-windows-amd64.exe
```

## Configuration

### Config File Location

Meowmail stores its configuration in:

- **macOS**: `~/Library/Application Support/meowmail/config.toml`
- **Linux**: `~/.config/meowmail/config.toml`
- **Windows**: `%APPDATA%\meowmail\config.toml`

### Database Location

- **macOS**: `~/Library/Application Support/meowmail/meowmail.db`
- **Linux**: `~/.local/share/meowmail/meowmail.db`
- **Windows**: `%APPDATA%\meowmail\meowmail.db`

### Manual Configuration

Create a config file at the location above with:

```toml
[general]
theme = "light"  # or "dark"
startup_check_mail = true
sync_interval_minutes = 5

[gmail]
# OAuth credentials (get from Google Cloud Console)
client_id = "your-client-id.apps.googleusercontent.com"
client_secret = "your-client-secret"

[outlook]
# OAuth credentials (get from Azure Portal)
client_id = "your-azure-client-id"
client_secret = "your-azure-client-secret"

[aftersmtp]
default_gateway = "amp.msgs.global:4433"
enable_blockchain_proofs = true

[mailblocks]
ipfs_gateway = "https://ipfs.io"
ethereum_rpc = "https://mainnet.infura.io/v3/YOUR-PROJECT-ID"
```

## First-Time Setup

### 1. Launch Meowmail

```bash
meowmail
```

The GUI will open automatically.

### 2. Add Your First Account

**For Gmail:**
1. Tools → Manage Accounts → Add Account
2. Select "Gmail"
3. Click "Authenticate with OAuth"
4. Browser opens → Sign in with Google → Grant permissions
5. Return to Meowmail → Account added!

**For Outlook:**
1. Tools → Manage Accounts → Add Account
2. Select "Outlook"
3. Click "Authenticate with OAuth"
4. Browser opens → Sign in with Microsoft → Grant permissions
5. Return to Meowmail → Account added!

**For AfterSMTP/msgs.global:**
1. Tools → Migration Wizard (or Manage Accounts)
2. Create New DID Identity
3. Choose username (e.g., `yourname`)
4. Keys generated automatically
5. DID created: `did:aftersmtp:msgs.global:yourname`

**For IMAP:**
1. Tools → Manage Accounts → Add Account
2. Select "IMAP"
3. Enter server details:
   - IMAP Host: `imap.example.com`
   - Port: `993`
   - Username: your email
   - Password: your password
4. Save → Messages sync automatically

### 3. Sync Your Email

- Messages will sync automatically
- Watch the sync status in the bottom-right
- First sync may take a few minutes

### 4. Send Your First Message

1. Click "Composer" tab
2. Select account (From dropdown)
3. Enter recipient
4. Write subject and message
5. Click "Send Message"

### 5. (Optional) Migrate from Gmail to AfterSMTP

1. Tools → Migration Wizard
2. Follow the step-by-step guide
3. Select Gmail as source
4. Select AfterSMTP as target
5. Choose folders to migrate
6. Run migration (~15-20 mins for 1000 messages)

## Troubleshooting

### macOS: "Cannot open meowmail because developer cannot be verified"

**Solution:**
```bash
# Option 1: Remove quarantine
xattr -d com.apple.quarantine /usr/local/bin/meowmail

# Option 2: System Preferences
# Go to Security & Privacy → General → Click "Open Anyway"
```

### Linux: "error while loading shared libraries"

**Solution:**
```bash
# Ubuntu/Debian
sudo apt install libgl1-mesa-glx libxi6 libxrandr2 libxcursor1 libxinerama1

# Fedora/RHEL
sudo dnf install mesa-libGL libXi libXrandr libXcursor libXinerama
```

### Windows: "This app can't run on your PC"

**Solution:**
- Download the correct architecture (amd64 for 64-bit, 386 for 32-bit)
- Install Visual C++ Redistributable: https://aka.ms/vs/17/release/vc_redist.x64.exe

### "Failed to connect to Gmail API"

**Solution:**
1. Check OAuth credentials in config
2. Ensure redirect URI is set to: `http://localhost:8080/oauth2callback`
3. Enable Gmail API in Google Cloud Console
4. Regenerate OAuth token: Tools → Accounts → Re-authenticate

### "Failed to connect to AfterSMTP gateway"

**Solution:**
1. Check network connection
2. Verify gateway URL: `amp.msgs.global:4433`
3. Check firewall allows outbound connections on port 4433
4. Try alternative gateway if available

### "Database is locked"

**Solution:**
1. Close all Meowmail instances
2. Remove lock file:
   ```bash
   # macOS/Linux
   rm ~/Library/Application\ Support/meowmail/meowmail.db-lock

   # Windows
   del %APPDATA%\meowmail\meowmail.db-lock
   ```

### Build Errors

**"protoc-gen-go: program not found"**
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH=$PATH:$(go env GOPATH)/bin
```

**"cannot find package"**
```bash
go mod download
go mod tidy
```

## Advanced Configuration

### Running as a Background Service

**macOS (launchd):**
```bash
# Create ~/Library/LaunchAgents/com.meowmail.daemon.plist
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.meowmail.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/meowmail</string>
        <string>daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>

# Load the service
launchctl load ~/Library/LaunchAgents/com.meowmail.daemon.plist
```

**Linux (systemd):**
```bash
# Create ~/.config/systemd/user/meowmail.service
[Unit]
Description=Meowmail Email Client Daemon
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/meowmail daemon
Restart=on-failure

[Install]
WantedBy=default.target

# Enable and start
systemctl --user enable meowmail
systemctl --user start meowmail
```

**Windows (Task Scheduler):**
1. Open Task Scheduler
2. Create Task → "Meowmail Daemon"
3. Trigger: At log on
4. Action: Start `C:\Program Files\Meowmail\meowmail.exe daemon`
5. Save

### Environment Variables

```bash
# Override config location
export MEOWMAIL_CONFIG_DIR=/custom/path

# Enable debug logging
export MEOWMAIL_DEBUG=1

# Set custom database path
export MEOWMAIL_DB_PATH=/custom/meowmail.db

# Disable auto-update check
export MEOWMAIL_NO_UPDATE_CHECK=1
```

## Uninstallation

### macOS

```bash
# Remove binary
sudo rm /usr/local/bin/meowmail

# Remove config and data
rm -rf ~/Library/Application\ Support/meowmail
rm -rf ~/.config/meowmail

# Remove launchd service (if configured)
launchctl unload ~/Library/LaunchAgents/com.meowmail.daemon.plist
rm ~/Library/LaunchAgents/com.meowmail.daemon.plist
```

### Linux

```bash
# Remove binary
sudo rm /usr/local/bin/meowmail

# Remove config and data
rm -rf ~/.config/meowmail
rm -rf ~/.local/share/meowmail

# Remove systemd service (if configured)
systemctl --user stop meowmail
systemctl --user disable meowmail
rm ~/.config/systemd/user/meowmail.service
```

### Windows

```powershell
# Remove from Program Files
Remove-Item "C:\Program Files\Meowmail" -Recurse

# Remove config and data
Remove-Item "$env:APPDATA\meowmail" -Recurse

# Remove from PATH (if added)
# System Properties → Environment Variables → Edit Path → Remove entry
```

## Getting Help

- **Documentation**: https://docs.meowmail.dev
- **GitHub Issues**: https://github.com/ryan/meowmail/issues
- **Community Forum**: https://community.meowmail.dev
- **Email**: support@meowmail.dev

---

**Last Updated**: 2026-03-16
**Version**: 1.0.0
