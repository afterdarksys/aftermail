package web3mail

import (
	"fmt"
)

// GroupKeyStore represents the securely shared symmetric key given to members of an encrypted web3 messaging group
type GroupKeyStore struct {
	GroupID     string
	OwnerDID    string
	SharedKey   []byte
	AdminsDIDs  []string
	MembersDIDs []string
}

// AMPGroupPayload encapsulates an AfterSMTP Message targeted at a shared-key group 
type AMPGroupPayload struct {
	GroupID       string
	Ciphertext    []byte
	SenderDID     string
	SenderSig     []byte
}

// CreateGroup initializes a new GroupKeyStore mathematically and mints a single AES-GCM shared key
func CreateGroup(name string, ownerDID string) *GroupKeyStore {
	// STUB: Wrap crypto/rand generating a persistent symmetric key.
	// Store it securely in the owner's Keychain.
	return &GroupKeyStore{
		GroupID:    name,
		OwnerDID:   ownerDID,
		AdminsDIDs: []string{ownerDID},
	}
}

// AddMember mints a copy of the GroupKeyStore's SharedKey and encrypts it using the new member's X25519 public credential
func (g *GroupKeyStore) AddMember(memberDID string, memberX25519PubKey []byte) ([]byte, error) {
   // STUB: Return the wrapped Group key. The receiver unwraps it and stores it locally. 
   return nil, fmt.Errorf("X25519 key encapsulation not implemented for group sharing")
}

// DecryptPayload unseals a group's inbound message utilizing the locally cached copy of the SharedKey
func (g *GroupKeyStore) DecryptPayload(payload *AMPGroupPayload) ([]byte, error) {
	if g.SharedKey == nil {
		return nil, fmt.Errorf("local agent missing active shared key for group %s", g.GroupID)
	}
	// STUB: Apply AES-GCM unlocking routine on payload.Ciphertext
	return nil, fmt.Errorf("decryption routine missing")
}
