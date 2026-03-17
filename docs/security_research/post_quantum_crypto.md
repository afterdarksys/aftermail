# Post-Quantum Cryptography (PQC) Evaluation

## Objective
Deprecate traditional elliptic curve cryptography (`X25519` for Key Exchange, `Ed25519` for signatures) in the AfterMail spec before "Store Now, Decrypt Later" quantum attacks breach the web3 registry.

## Investigation & Feasibility
1. **NIST Standardization:** The primary targets for adoption are **Kyber** (ML-KEM) for Key Encapsulation and **Dilithium** (ML-DSA) for Digital Signatures.
2. **Implementation Strategy:**
   - Import `github.com/cloudflare/circl`, Cloudflare's Go cryptographic library containing optimized assembly for PQC primitives.
   - **Mailblocks Registry:** Upgrade the Ethereum Smart Contract to store Dilithium Public Keys instead of 32-byte Ed25519 strings. (Note: Dilithium keys are significantly larger, requiring Layer 2 rollups to maintain cheap gas fees).
   - **AMP Headers:** Expand the `AMPMessage` payload structs to encapsulate `Kyber1024` ciphertext capsules (1568 bytes) rather than tiny AES-GCM tags.

## Conclusion
PQC is ready for immediate experimental deployment utilizing `circl`. We will implement a feature-flag inside Fyne `Settings -> Security` to emit `PQC-Only` envelopes.
