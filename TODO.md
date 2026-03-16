# ADS Mail (codename: meowmail) TODO List

## High Priority (v1.1)

### Core Functionality
- [ ] Complete OAuth2 token refresh logic for Gmail
- [ ] Complete OAuth2 token refresh logic for Outlook
- [ ] Implement actual message fetching from IMAP servers
- [ ] Implement actual message sending via SMTP
- [ ] Add local database storage for accounts
- [ ] Add local database storage for messages
- [ ] Implement message caching and sync logic
- [ ] Add attachment download functionality
- [ ] Add attachment upload functionality

### AfterSMTP Integration
- [ ] Complete X25519 key exchange implementation
- [ ] Complete AES-GCM encryption/decryption
- [ ] Integrate with actual AfterSMTP gateway
- [ ] Implement blockchain ledger queries for DID verification
- [ ] Add proper Ed25519 signature verification
- [ ] Complete QUIC transport layer
- [ ] Add connection pooling for gRPC connections

### Mailblocks Integration
- [ ] Implement Ethereum wallet integration
- [ ] Add IPFS client for distributed storage
- [ ] Implement stake threshold configuration
- [ ] Add quarantine review interface
- [ ] Implement stake acceptance/rejection flow

### Migration Wizard
- [ ] Actually execute migration (currently mock)
- [ ] Add progress tracking with database
- [ ] Implement resume capability for interrupted migrations
- [ ] Add rollback functionality
- [ ] Handle migration errors gracefully
- [ ] Add conflict resolution (duplicate messages)

### Security
- [ ] Implement OS keychain integration for key storage
  - macOS: Keychain Access
  - Windows: DPAPI
  - Linux: Secret Service API
- [ ] Add password encryption at rest
- [ ] Implement OAuth token encryption
- [ ] Add private key encryption (Ed25519, X25519)
- [ ] Implement certificate pinning for AfterSMTP
- [ ] Add DANE/DNSSEC verification

## Medium Priority (v1.2)

### UI/UX Improvements
- [ ] Add message threading view
- [ ] Implement search functionality
- [ ] Add folder management (create, delete, rename)
- [ ] Implement drag-and-drop for messages
- [ ] Add keyboard shortcuts
- [ ] Implement message filters/rules UI
- [ ] Add notification system
- [ ] Implement dark mode theme
- [ ] Add account switcher in UI

### Contact Management
- [ ] Create contacts database schema
- [ ] Implement contact import from Gmail
- [ ] Implement contact import from Outlook
- [ ] Add contact editing interface
- [ ] Implement contact groups
- [ ] Add contact sync between accounts

### Composer Enhancements
- [ ] Add rich text editor (WYSIWYG)
- [ ] Implement HTML email templates
- [ ] Add inline image support
- [ ] Implement draft auto-save
- [ ] Add spell check
- [ ] Implement email signatures
- [ ] Add CC/BCC fields

### Message Viewer
- [ ] Proper HTML rendering (use webview)
- [ ] Add print functionality
- [ ] Implement "view raw" mode
- [ ] Add message export (EML, PDF)
- [ ] Implement inline attachment preview
- [ ] Add reply/forward functionality

## Lower Priority (v2.0+)

### Advanced Features
- [ ] Calendar integration (CalDAV)
- [ ] Task management integration
- [ ] Note-taking capability
- [ ] Encrypted group messaging
- [ ] Video call integration
- [ ] Screen sharing
- [ ] File sharing with encryption

### Protocol Extensions
- [ ] PGP/GPG compatibility layer
- [ ] S/MIME support
- [ ] Multi-signature workflows
- [ ] Smart contract triggered emails
- [ ] Automated response system

### Platform Support
- [ ] Mobile apps (iOS/Android via React Native)
- [ ] Browser extension
- [ ] Web app version
- [ ] CLI improvements for automation
- [ ] API for third-party integrations

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
- [ ] CI/CD pipeline setup
- [ ] Automated builds for all platforms
- [ ] Code signing for macOS/Windows
- [ ] Auto-update mechanism
- [ ] Crash reporting
- [ ] Analytics (privacy-preserving)

## Bug Fixes

### Known Issues
- [ ] Fix duplicate library warning during build
- [ ] Handle network timeouts gracefully
- [ ] Fix race conditions in message sync
- [ ] Handle large attachment files (>100MB)
- [ ] Fix memory leaks in GUI
- [ ] Handle malformed MIME messages
- [ ] Fix timezone handling in message dates

## Research & Investigation

### Performance
- [ ] Profile message parsing performance
- [ ] Optimize database queries
- [ ] Reduce memory usage
- [ ] Improve startup time
- [ ] Lazy load large message lists

### Security Research
- [ ] Investigate homomorphic encryption for search
- [ ] Research zero-knowledge proofs for DID
- [ ] Evaluate post-quantum cryptography options
- [ ] Study secure multi-party computation

### Standards Compliance
- [ ] Full SMTP RFC compliance testing
- [ ] IMAP IDLE support
- [ ] IMAP Extensions support
- [ ] OAuth 2.1 migration
- [ ] OpenID Connect integration

## Nice to Have

- [ ] Plugin system for extensions
- [ ] Custom themes support
- [ ] Accessibility improvements (screen readers)
- [ ] Internationalization (i18n)
- [ ] Offline mode
- [ ] Backup/restore functionality
- [ ] Import from other email clients (Thunderbird, Apple Mail)
- [ ] Export to other formats
- [ ] Message templates
- [ ] Scheduled sending
- [ ] Read receipts
- [ ] Email tracking prevention
- [ ] Spam filter training
- [ ] Phishing detection

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
