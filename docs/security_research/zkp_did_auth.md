# Zero-Knowledge Proofs (ZKP) for DID Identity

## Objective
Prove possession of an AfterSMTP Mailblocks DID (Decentralized Identifier) and authorization to dispatch an email without revealing the underlying private `Ed25519` signature key to the relay gateway.

## Investigation & Feasibility
1. **The Problem:** Currently, AfterMail attaches a plaintext `Ed25519` cryptographic signature to the `AMP.SenderSig` header. The AfterSMTP gateway uses `go-ethereum` to verify the sender's public key against the Mailblocks registry.
2. **The ZK Solution:** Implement `gnark` (ConsenSys Go ZK-SNARK library) inside `pkg/security`.
3. **The Circuit:**
   - **Public Inputs:** The Web3 Sender DID string (`did:aftersmtp:ryan`), the Hash of the Email Body, and the current Ethereum Block Root.
   - **Private (Witness) Inputs:** The user's underlying `Ed25519` private key.
   - **The Proof:** The sender generates a succinct proof ($\pi$) locally on their machine stating: "I know the private key that maps to this DID in the current Ethereum state, and I sign this email hash".
   
## Conclusion
Integrating `gnark` into the `AfterSMTP` envelope allows true anonymous relaying. The SMTP gateway verifies the SNARK proof in milliseconds without ever seeing the mathematical signature, completely neutralizing physical key-extraction attacks on the wire. This is strongly slated for production inside `pkg/web3mail/zkp.go`.
