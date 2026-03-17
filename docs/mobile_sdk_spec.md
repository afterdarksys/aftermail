# AfterMail React Native SDK Specification

## Overview
The mobile application (iOS/Android) bridges to the local desktop AfterMail network via pairing codes, transforming smartphones into verifiable Identity Signers and remote thin clients.

### 1. Pairing Protocol (QR Code)
- Desktop Fyne UI displays a QR Code encoding: `aftermail://pair?session=xyz&seed=base64...`
- The React Native Mobile App scans the QR code, exchanges Ed25519 public keys, and derives an AES-GCM session key mapping back to `127.0.0.1:4460` (tunneled securely if remote).

### 2. Thin Client Architecture
The mobile application is strictly stateless to protect local key materials.
- **REST Endpoints**: Calls `/api/v1/inbox` using an `Authorization: Bearer <JWT>` issued by the Desktop pairing process.
- **Push Notifications**: Relies on a decentralized Push Provider via WebSockets hooked into the Desktop QUIC tunnel.

### 3. Key React Native Modules
- `@aftermail/react-native-crypto`: C++ JSI bindings to implement the standard Go Curve25519 algorithms identically to desktop. 
- `@aftermail/react-native-amf`: AMF structure parser transforming the `.bytes` sequences into Mobile-native UI render blocks.
