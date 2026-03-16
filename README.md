# 🐱 Meowmail

**Making Email Purr Again** - The next-generation email client supporting traditional protocols AND cutting-edge Web3/blockchain-backed messaging.

## What is Meowmail?

Meowmail is a comprehensive email platform that bridges traditional email (IMAP, Gmail, Outlook) with next-generation protocols (AfterSMTP, Mailblocks). It's both a protocol debugging tool AND a full-featured email client with first-class support for encrypted, blockchain-verified messaging.

### The Strategy

Most Web3 projects fail because they require users to abandon existing systems. Meowmail takes a different approach:

1. **Download for traditional email** - Works perfectly with Gmail, Outlook, any IMAP server
2. **Discover advanced features** - Built-in migration wizard, protocol testing, security verification
3. **Graduate to Web3** - When ready, use AfterSMTP/Mailblocks for encrypted DID-based messaging
4. **Keep both** - Traditional and Web3 email coexist in one unified interface

This gives AfterSMTP and Mailblocks a **real product** with actual users, not just a token and whitepaper.

## Features

### Traditional Email Support

- **IMAP/POP3/SMTP** - Standard email protocols
- **Gmail Integration** - OAuth2 authentication, Gmail API support
- **Outlook Integration** - Microsoft Graph API for Outlook.com and Microsoft 365
- **Folder Management** - Inbox, Sent, Trash, custom folders
- **Attachments** - Full support for file attachments

### AfterSMTP Advanced Message Protocol (AMP)

- **DID-Based Identity** - No more passwords! `did:aftersmtp:msgs.global:username`
- **End-to-End Encryption** - X25519 + AES-GCM-256 encryption
- **Cryptographic Signatures** - Ed25519 signatures, blockchain-backed verification
- **Modern Message Format (AMF)** - Replaces legacy MIME with clean Protobuf structure
- **Proof of Transit** - Every message recorded on Substrate blockchain
- **QUIC Transport** - 0-RTT latency, faster than traditional SMTP
- **msgs.global Client** - Connect to the free public AfterSMTP gateway

### Mailblocks Web3 Email

- **Proof-of-Stake Spam Prevention** - Senders stake ETH to reach your inbox
- **IPFS Storage** - Distributed message storage
- **Quarantine System** - Review staked emails before accepting
- **Wallet Integration** - Ethereum wallet support

### Protocol Testing & Debugging

- **Grammar Testing** - SMTP, IMAP, POP3 compliance checks
- **Security Verification** - SPF, DKIM, DMARC, MTA-STS, BIMI, ARC
- **Raw Session Debugging** - Interactive protocol sessions
- **Message Inspector** - Parse and analyze both MIME and AMF messages
- **AfterSMTP Gateway Testing** - gRPC connection testing, DID verification

### Migration Tools

- **Migration Wizard** - Step-by-step guide to migrate from Gmail/Outlook to AfterSMTP
- **Bulk Import** - Import thousands of messages preserving metadata
- **Format Conversion** - Automatic MIME → AMF conversion
- **Contact Migration** - (Coming soon)

## Architecture

### Message Formats

**MIME (Traditional):**
```
From: sender@example.com
To: recipient@example.com
Subject: Hello
Content-Type: text/plain

Message body here
```

**AMF (AfterSMTP):**
```protobuf
message AMFPayload {
    string subject = 1;
    string text_body = 2;
    string html_body = 3;
    repeated Attachment attachments = 4;
    map<string, string> extended_headers = 5;
}
```

Benefits of AMF over MIME:
- ✅ Native binary format (no base64 overhead)
- ✅ Structured data (no complex parsing)
- ✅ Hash verification for attachments
- ✅ Clean separation of headers and body
- ✅ Extensible with custom headers

### Account Types

Meowmail supports 7 account types:

| Type | Protocol | Authentication | Use Case |
|------|----------|----------------|----------|
| IMAP | IMAP/SMTP | Username/Password | Traditional mail servers |
| POP3 | POP3/SMTP | Username/Password | Legacy systems |
| Gmail | Gmail API | OAuth2 | Google Workspace |
| Outlook | Graph API | OAuth2 | Microsoft 365 |
| msgs.global | AfterSMTP/gRPC | DID + Ed25519 | Free encrypted email |
| AfterSMTP | AfterSMTP/gRPC | DID + Ed25519 | Self-hosted gateway |
| Mailblocks | Web3 + IPFS | Ethereum Wallet | PoS email |

## Installation

```bash
# Clone the repository
git clone https://github.com/ryan/meowmail
cd meowmail

# Install dependencies
go mod tidy

# Build
go build -o meowmail

# Run GUI
./meowmail

# Or use CLI commands
./meowmail test smtp --host mail.example.com --port 25
./meowmail verify example.com
./meowmail amp send --target did:aftersmtp:msgs.global:alice --payload "Hello!"
```

## Configuration

### Setting up Gmail

1. Go to Google Cloud Console
2. Create OAuth2 credentials
3. Add to Meowmail: Tools → Manage Accounts → Add Gmail Account
4. Authenticate via OAuth

### Setting up AfterSMTP

1. Get a DID identity:
   - Use Migration Wizard (Tools → Migration Wizard)
   - Or CLI: `./meowmail register --username yourname`

2. Keys are automatically generated:
   - Ed25519 signing key
   - X25519 encryption key

3. Connect to msgs.global or self-hosted gateway

### Setting up Mailblocks

1. Connect Ethereum wallet
2. Configure IPFS endpoint
3. Set quarantine stake threshold

## Usage Examples

### Send Traditional Email

```bash
# Via CLI
./meowmail send --to user@example.com --subject "Hello" --body "Test message"

# Via GUI
Composer → Select IMAP account → Write message → Send
```

### Send AfterSMTP Encrypted Message

```bash
# Via CLI
./meowmail amp send \
  --did did:aftersmtp:msgs.global:ryan \
  --target did:aftersmtp:msgs.global:alice \
  --payload "Encrypted message"

# Via GUI
Composer → Select AfterSMTP account → Enter DID → Write message → Send
```

### Migrate from Gmail

```bash
# Via GUI: Tools → Migration Wizard
1. Authenticate with Gmail
2. Create/select AfterSMTP DID
3. Select folders to migrate
4. Run migration (converts MIME → AMF automatically)
```

### Inspect Message Format

```bash
# Via GUI: Protocol Inspector → Inspect Message
# Paste MIME or AMF hex → Auto-detects format → Shows parsed structure
```

## Protocol Comparison

| Feature | SMTP/MIME | AfterSMTP/AMF | Mailblocks |
|---------|-----------|---------------|------------|
| Encryption | TLS only (transport) | E2E (X25519+AES) | E2E + IPFS |
| Signatures | DKIM (optional) | Ed25519 (required) | Blockchain |
| Identity | Email address | DID (blockchain) | Ethereum address |
| Spam Prevention | Filters | DID reputation | Proof-of-stake |
| Format | Text (MIME) | Protobuf (AMF) | Protobuf + IPFS |
| Storage | Centralized | Gateway-based | IPFS |
| Proof of Delivery | Bounce messages | Blockchain receipt | Smart contract |

## Development

### Project Structure

```
meowmail/
├── cmd/                    # CLI commands
│   ├── root.go            # Main command
│   ├── test.go            # Protocol testing
│   ├── verify.go          # Security verification
│   ├── amp.go             # AfterSMTP commands
│   └── web3.go            # Mailblocks commands
├── internal/
│   ├── gui/               # Fyne GUI components
│   │   ├── gui.go         # Main window
│   │   ├── composer.go    # Email composer
│   │   ├── folders.go     # Folder/inbox view
│   │   ├── amfviewer.go   # AMF message viewer
│   │   ├── migration.go   # Migration wizard
│   │   └── protocol.go    # Protocol inspector
│   └── daemonapi/         # Background daemon
└── pkg/
    ├── accounts/          # Account management
    │   ├── types.go       # Account/Message types
    │   ├── gmail.go       # Gmail OAuth client
    │   ├── outlook.go     # Outlook Graph client
    │   └── msgsglobal.go  # AfterSMTP client
    ├── amp/               # AfterSMTP client library
    ├── web3mail/          # Mailblocks client library
    ├── security/          # SPF/DKIM/DMARC verification
    ├── smtp/              # SMTP testing
    ├── imap/              # IMAP testing
    └── pop3/              # POP3 testing
```

### Building from Source

```bash
# Install Go 1.21+
go version

# Install dependencies
go mod download

# Build for your platform
go build -o meowmail

# Build for all platforms
GOOS=windows GOARCH=amd64 go build -o meowmail.exe
GOOS=darwin GOARCH=amd64 go build -o meowmail-mac
GOOS=linux GOARCH=amd64 go build -o meowmail-linux
```

## Roadmap

### v1.0 (Current)
- ✅ Traditional IMAP/POP3/SMTP support
- ✅ Gmail OAuth integration
- ✅ Outlook Graph API integration
- ✅ AfterSMTP AMP client (msgs.global)
- ✅ Mailblocks Web3 client
- ✅ Migration wizard
- ✅ Protocol testing tools
- ✅ AMF message viewer

### v1.1 (Next)
- ⬜ Contact management and migration
- ⬜ Attachment encryption for AMF
- ⬜ Calendar integration
- ⬜ Push notifications
- ⬜ Mobile apps (React Native with AMP support)

### v2.0 (Future)
- ⬜ Full HTML email rendering
- ⬜ PGP compatibility layer
- ⬜ Multi-signature workflows
- ⬜ Smart contract triggered emails
- ⬜ Decentralized group messaging

## Contributing

We welcome contributions! This project bridges traditional email and Web3 - opportunities exist in:

- Protocol implementations
- UI/UX improvements
- Security audits
- Documentation
- Testing
- Client libraries for other languages

## Security

### Responsible Disclosure

Report security vulnerabilities to: security@meowmail.dev

### Cryptography

- **Ed25519** - Message signatures (libsodium)
- **X25519** - Key exchange (libsodium)
- **AES-GCM-256** - Payload encryption
- **TLS 1.3** - Transport security
- **DANE** - Certificate pinning

### Key Storage

Private keys are stored encrypted at rest using OS keychain:
- macOS: Keychain Access
- Windows: DPAPI
- Linux: Secret Service API

## License

MIT License - See LICENSE file

## Credits

Built with:
- [Fyne](https://fyne.io) - Cross-platform GUI
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [AfterSMTP](https://github.com/aftersmtp/aftersmtp) - AMP protocol
- [go-imap](https://github.com/emersion/go-imap) - IMAP client
- [go-msgauth](https://github.com/emersion/go-msgauth) - DKIM/SPF/DMARC

---

**Making email purr again** 🐱📧

For support, visit: https://meowmail.dev
```

This README gives users and developers a complete picture of what Meowmail is and why it matters - it's not just another email client, it's the real-world product that makes AfterSMTP and Mailblocks actually useful!
