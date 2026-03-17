# ADS Mail (codename: meowmail) TODO List

## High Priority (v1.1)

### Core Functionality
- [x] Basic email rendering (HTML and Plain text)
- [x] Folder management (Inbox, Sent, Archive, Trash)
- [x] Advanced search capabilities
- [x] Offline support with local caching
- [x] Implement actual message fetching from IMAP servers
- [x] Implement actual message sending via SMTP
- [x] Add local database storage for accounts
- [x] Add local database storage for messages
- [x] Implement message caching and sync logic
- [x] Add attachment download functionality
- [x] Add attachment upload functionality

### AfterSMTP Integration
- [x] Complete X25519 key exchange implementation
- [x] Complete AES-GCM encryption/decryption
- [x] Integrate with actual AfterSMTP gateway
- [x] Implement blockchain ledger queries for DID verification
- [x] Add proper Ed25519 signature verification
- [x] Complete QUIC transport layer
- [x] Add connection pooling for gRPC connections

### Mailblocks Integration
- [x] Implement Ethereum wallet integration
- [ ] Add IPFS client for distributed storage
- [ ] Implement stake threshold configuration
- [ ] Add quarantine review interface
- [ ] Implement stake acceptance/rejection flow

### Migration Wizard
- [x] Actually execute migration (currently mock)
- [x] Add progress tracking with database
- [x] Implement resume capability for interrupted migrations
- [x] Add rollback functionality
- [x] Handle migration errors gracefully
- [x] Add conflict resolution (duplicate messages)

### Security
- [x] Implement OS keychain integration for key storage
  - macOS: Keychain Access
  - Windows: DPAPI
  - Linux: Secret Service API
- [x] Add password encryption at rest
- [x] Implement OAuth token encryption
- [x] Add private key encryption (Ed25519, X25519)
- [ ] Implement certificate pinning for AfterSMTP
- [ ] Add DANE/DNSSEC verification

## Medium Priority (v1.2)

### UI/UX Improvements
- [x] Add message threading view
- [x] Implement search functionality
- [x] Add folder management (create, delete, rename)
- [x] Implement drag-and-drop for messages
- [x] Add keyboard shortcuts
- [x] Implement message filters/rules UI
- [x] Add notification system
- [x] Implement dark mode theme
- [x] Add account switcher in UI

### Contact Management
- [x] Create contacts database schema
- [x] Implement contact import from Gmail
- [x] Implement contact import from Outlook
- [x] Add contact editing interface
- [x] Implement contact groups
- [x] Add contact sync between accounts

### Composer Enhancements
- [x] Add Cc/Bcc support
- [x] Implement rich text editing (basic formatting)
- [x] Add email signatures support
- [x] Support draft auto-saving
- [x] Add email templates
- [x] Add spell check

### Message Viewer
- [x] Proper HTML rendering (use webview)
- [x] Add print functionality
- [x] Implement "view raw" mode
- [x] Add message export (EML, PDF)
- [x] Implement inline attachment preview
- [x] Add reply/forward functionality

## Lower Priority (v2.0+)

### Advanced Features
- [x] Calendar integration (CalDAV)
- [x] Task management integration
- [x] Note-taking capability
- [x] Encrypted group messaging
- [x] Video call integration
- [x] Screen sharing
- [x] File sharing with encryption

### Protocol Extensions
- [x] PGP/GPG compatibility layer
- [x] S/MIME support
- [x] Multi-signature workflows
- [x] Smart contract triggered emails
- [x] Automated response system

### Platform Support
- [x] Mobile apps (iOS/Android via React Native)
- [x] Browser extension
- [x] Web app version
- [x] CLI improvements for automation
- [x] API for third-party integrations

### Testing & Quality
- [ ] Add unit tests for all packages
- [ ] Add integration tests
- [ ] Add E2E tests for GUI
- [ ] Performance benchmarks
- [ ] Load testing for AfterSMTP client
- [ ] Security audit
- [ ] Code coverage > 80%

### Documentation
- [ ] API documentation
- [ ] Developer guide
- [ ] User manual
- [ ] Video tutorials
- [ ] Architecture diagrams
- [ ] Security best practices guide

### DevOps
- [x] CI/CD pipeline setup
- [x] Automated builds for all platforms
- [x] Code signing for macOS/Windows
- [x] Auto-update mechanism
- [x] Crash reporting
- [x] Analytics (privacy-preserving)

## Bug Fixes

### Known Issues
- [x] Fix duplicate library warning during build
- [x] Handle network timeouts gracefully
- [x] Fix race conditions in message sync
- [ ] Handle large attachment files (>100MB)
- [ ] Fix memory leaks in GUI
- [x] Handle malformed MIME messages
- [ ] Fix timezone handling in message dates

## Research & Investigation

### Performance
- [x] Profile message parsing performance
- [x] Optimize database queries
- [x] Reduce memory usage
- [x] Improve startup time
- [x] Lazy load large message lists

### Security Research
- [x] Investigate homomorphic encryption for search
- [x] Research zero-knowledge proofs for DID
- [x] Evaluate post-quantum cryptography options
- [x] Study secure multi-party computation

### Standards Compliance
- [x] Full SMTP RFC compliance testing
- [x] IMAP IDLE support
- [x] IMAP Extensions support
- [x] OAuth 2.1 migration
- [x] OpenID Connect integration

## All Nice to Have Features

- [x] Plugin system for extensions
- [ ] Custom themes support (`theme.go`)
- [ ] Accessibility improvements (screen readers)
- [ ] Internationalization (i18n)
- [x] Offline mode integration (SQLite caching)
- [x] Backup/restore functionality (SQLite dumping)
- [x] Import from other email clients (`.mbox` parser)
- [x] Consolidate Message export to PDF/EML
- [ ] User-customizable Message templates
- [x] Scheduled sending dispatcher engine
- [x] Read receipts
- [x] Email tracking prevention
- [x] Spam filter training
- [x] Phishing detection

## Community & Marketing

- [ ] Create project website
- [ ] Set up community forum
- [ ] Create Discord/Slack channel
- [ ] Write blog posts about Web3 email
- [ ] Create comparison charts (ADS Mail vs traditional)
- [ ] Develop marketing materials
- [ ] Create demo videos
- [ ] Submit to product directories
- [ ] Engage with crypto/Web3 communities
- [ ] Partner with AfterSMTP ecosystem projects

## Infrastructure

- [ ] Set up public AfterSMTP gateway for testing
- [ ] Create public Mailblocks test network
- [ ] Set up documentation site
- [ ] Create issue tracker
- [ ] Set up discussions forum
- [ ] Create donation/sponsorship page
- [ ] Set up automated release process

---

**Priority Legend:**
- High: Essential for v1.1 release
- Medium: Important but not blocking
- Lower: Future enhancements

**Status Tracking:**
- [ ] Not started
- [~] In progress
- [x] Completed
- [!] Blocked

Last updated: 2026-03-16
