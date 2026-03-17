# AfterMail Browser Extension specification

## Overview
The Google Chrome and Mozilla Firefox extensions interact directly with the `127.0.0.1:4460` local REST/gRPC hybrid daemon. It allows signing emails transparently in the browser and managing DIDs without opening the Fyne UI.

### Permissions Required
- `"nativeMessaging"`: Allows bridging Native Binary JSON payloads to the `aftermaild` root service.
- `"activeTab"`: Extracts `mailto:` links dynamically to inject the AfterMail Web-Composer directly in-page.

### Injection Flow
1. **Background Service Worker**: Loops an HTTP ping to `http://127.0.0.1:4460/api/v1/status` identifying the state of the local daemon.
2. **MailTo Hook**: If a user clicks an email address on the web, `contentscript.js` suppresses the default OS fallback and triggers a Modal Overlay embedding the `/` Dashboard Composer route.
3. **Key Signing**: When the user approves a message inside the extension popup, `chrome.runtime.sendNativeMessage` is piped to the local Go executable for raw `Ed25519` mathematical verification.
