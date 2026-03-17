# Homomorphic Encryption (FHE) for Email Search

## Objective
Enable full-text search over the local SQLite `messages` and `notes` tables without ever retaining plaintext decryption keys in RAM for extended periods.

## Investigation & Feasibility
1. **Current State:** AfterMail decrypts AMF `AES-GCM` payloads immediately into memory for rendering and Fyne list binding.
2. **Homomorphic Approach:** Utilizing a scheme like BGV or CKKS (via libraries like OpenFHE or Microsoft SEAL bindings for Go).
3. **Application:**
   - When a user receives an encrypted AMF message, the Mailblocks daemon parses the headers and *homomorphically encrypts* the tokenized body content into a secondary SQLite FTS5 (Full Text Search) index.
   - The user inputs a Search Query: `query = "invoice"`. 
   - The GUI homomorphically encrypts the search query, executes an algebraic evaluation against the FHE FTS5 index, and returns matching Row IDs.
   - The Row IDs correspond to the heavily encrypted base AMF blobs which are then individually decrypted.

## Conclusion
Fully Homomorphic Encryption (FHE) is mathematically viable but computationally heavy for Go-native desktop SQLite bindings. The v2.0 implementation will focus on partial (Property-Preserving) encryption or Searchable Symmetric Encryption (SSE) instead of pure FHE to maintain UI responsiveness.
