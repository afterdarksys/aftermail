# AfterMail Features

## Repository Rename Complete
- **Old Name:** meowmail
- **New Name:** aftermail
- **GitHub:** https://github.com/afterdarksys/aftermail
- **Module:** github.com/afterdarksys/aftermail

## Build System
The `build.sh` script provides comprehensive build management:

```bash
./build.sh build           # Build all components
./build.sh build-gui       # Build GUI only
./build.sh build-daemon    # Build daemon only
./build.sh clean           # Clean artifacts
./build.sh install         # Install to system
./build.sh dev             # Quick dev build
./build.sh release         # Optimized release
./build.sh cross           # Cross-compile
```

## Mailblocks Wallet Integration

### Full Ethereum Wallet (`pkg/wallet/ethereum.go`)
- ✅ Create new wallets with ECDSA key pairs
- ✅ Import from private key (hex)
- ✅ Import/export keystore files
- ✅ Sign transactions and messages
- ✅ Verify signatures
- ✅ Balance formatting and parsing

### Mailblocks Client (`pkg/wallet/mailblocks.go`)
- ✅ Connect to Ethereum RPC nodes
- ✅ Query wallet balances (ETH and staked)
- ✅ **Stake ETH** to send emails
- ✅ **Refund stakes** for accepted emails
- ✅ **Slash stakes** for rejected spam
- ✅ Query quarantined emails
- ✅ Smart contract interaction ready

## IPFS Integration (`pkg/ipfs/`)
- ✅ Upload messages to IPFS
- ✅ Download by CID
- ✅ Pin/unpin management
- ✅ Health checks
- ✅ Support for distributed storage

## AI Assistant (`pkg/ai/`)

### Supported Providers
- **Anthropic (Claude)** - Default: claude-sonnet-4-20250514
- **OpenRouter** - Access to multiple models

### Features
- ✅ **Spell Checking** - AI-powered spelling correction
- ✅ **Grammar Checking** - Advanced grammar analysis
- ✅ **Improve Writing** - Enhance clarity and impact
- ✅ **Make Concise** - Shorten while preserving meaning
- ✅ **Make Formal** - Professional tone conversion
- ✅ **Make Friendly** - Casual tone conversion
- ✅ **Generate Draft** - Create emails from descriptions
- ✅ **Summarize** - Extract key points from emails

### BYOK (Bring Your Own Key)
All AI features use your own API keys - stored locally, never shared.

## GUI Features

### New "AfterSMTP/Web3" Tab
Four comprehensive sections:

#### 1. Wallet Management
- Create new Ethereum wallets
- Import from private key or keystore
- View address and balance
- Export private keys (with warnings)
- Transaction history display

#### 2. Mailblocks Integration
- **Quarantine Management**
  - View all staked emails awaiting review
  - Accept emails (refund stake to sender)
  - Reject as spam (slash stake)
  - View messages on IPFS
- **Statistics Dashboard**
  - Quarantined email count
  - Total staked amount
  - 24h accept/slash counts
  - Earnings from slashed spam
- **Settings**
  - Minimum stake threshold
  - Auto-accept from trusted contacts
  - Auto-slash known spammers

#### 3. IPFS Operations
- Connection status indicator
- Configure API endpoint
- Pin/unpin messages
- Upload to IPFS
- Fetch by CID
- Health monitoring

#### 4. AfterSMTP Gateway
- Gateway connection status
- DID display and management
- Create new DIDs
- Import existing DIDs
- Configure gateway URL
- Connection statistics

### Enhanced Composer
New AI Toolbar with buttons:
- **✓ Spell Check** - Check spelling
- **✓ Grammar** - Check grammar
- **🤖 AI Assistant** - Dropdown menu:
  - Improve Writing
  - Make Concise
  - Make Formal
  - Make Friendly
  - Generate Draft
  - Summarize

### Comprehensive Settings Dialog

#### General Tab
- Theme selection (OS, Dark, Light, Neon, Custom)
- Language selection
- Enable/disable spell checking
- Enable/disable grammar checking

#### AI Assistant Tab
- Provider selection (Anthropic/OpenRouter)
- API key input (password field)
- Model selection (optional)
- Test connection button
- Usage instructions and help text

#### Accounts Tab
- List all email accounts
- Add new accounts (IMAP, Gmail, Outlook, AfterSMTP, Mailblocks)
- Edit/remove existing accounts

#### Advanced Tab
- Data directory configuration
- Debug logging toggle
- Export/import data
- Clear cache

## Architecture

### Package Organization
```
aftermail/
├── cmd/
│   ├── aftermaild/        # Daemon binary
│   └── *.go              # CLI commands
├── internal/
│   ├── gui/              # All GUI components
│   │   ├── web3.go       # NEW: Web3/Wallet UI
│   │   ├── settings.go   # NEW: Comprehensive settings
│   │   └── ...
│   └── daemonapi/        # Daemon API
├── pkg/
│   ├── wallet/           # NEW: Ethereum & Mailblocks
│   ├── ipfs/             # NEW: IPFS client
│   ├── ai/               # NEW: AI assistant
│   ├── amp/              # AfterSMTP client
│   ├── web3mail/         # Mailblocks API
│   ├── accounts/         # Account management
│   ├── security/         # SPF/DKIM/DMARC
│   └── ...
└── build.sh              # NEW: Build script
```

## Integration with Mailblocks Backend

AfterMail includes a replace directive in `go.mod`:
```go
replace github.com/mailblocks/backend => /Users/ryan/development/mailblocks.io/backend
```

This allows AfterMail to use Mailblocks backend packages directly when available.

## Next Steps

### To Use Wallet Features:
1. Open AfterMail
2. Go to "AfterSMTP/Web3" tab → "Wallet" section
3. Create or import a wallet
4. Configure RPC endpoint in settings (optional)

### To Use AI Features:
1. Go to Settings → AI Assistant
2. Select provider (Anthropic or OpenRouter)
3. Enter your API key
4. Save settings
5. Use AI buttons in Composer

### To Use Mailblocks:
1. Set up wallet (see above)
2. Go to "AfterSMTP/Web3" tab → "Mailblocks"
3. Configure stake threshold
4. Review quarantined emails
5. Accept (refund) or Reject (slash)

### To Use IPFS:
1. Start IPFS daemon: `ipfs daemon`
2. Go to "AfterSMTP/Web3" tab → "IPFS"
3. Test connection
4. Upload/fetch messages

## Development

Build from source:
```bash
git clone https://github.com/afterdarksys/aftermail
cd aftermail
./build.sh build
./bin/aftermail  # Run GUI
./bin/aftermaild # Run daemon
```

## Links
- Repository: https://github.com/afterdarksys/aftermail
- Mailblocks: ~/development/mailblocks.io
- AfterSMTP: ~/development/aftersmtp
