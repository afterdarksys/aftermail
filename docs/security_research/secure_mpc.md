# Secure Multi-Party Computation (SMPC)

## Objective
Require distributed quorum consensus before an email can legally be decrypted or dispatched from the client.

## Investigation & Feasibility
1. **The Scenario:** An enterprise team owns a shared inbox (`team@aftersmtp.local`). The private key should never exist on a single machine.
2. **SMPC Mechanism:** Utilize **Threshold ECDSA / EdDSA**.
   - The private signing key is split into $N$ mathematical shares (e.g., 5 shares for 5 employees).
   - To dispatch an email, at least $T$ (e.g., 3 out of 5) employees must partially sign the AMF payload locally.
   - The AfterMail daemon opens a temporary libP2P QUIC stream to coordinate the Multi-Party Computation loop over the network.
   - The final signature is mathematically assembled without the master private key ever being reconstructed in memory.

## Conclusion
SMPC is highly dependent on our implementation of `pkg/web3mail/groupmail.go`. We will utilize Shamir's Secret Sharing initially for key recovery, before transitioning to a full threshold protocol for real-time dispatch authorization.
