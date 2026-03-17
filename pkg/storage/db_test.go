package storage

import (
	"testing"

	"github.com/afterdarksys/aftermail/pkg/accounts"
)

func TestDBStorage(t *testing.T) {
	// Use an in-memory SQLite database for testing CRUD operations
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize test DB: %v", err)
	}
	defer db.Close()

	// 1. Account CRUD
	acc := &accounts.Account{
		Name:          "Test IMAP",
		Type:          accounts.TypeIMAP,
		Email:         "test@example.com",
		ImapHost:      "imap.example.com",
		ImapPort:      993,
		ImapUseTLS:    true,
		DID:           "did:aftersmtp:msgs.global:ryan",
		WalletAddress: "0x123",
		Enabled:       true,
	}

	id, err := db.InsertAccount(acc)
	if err != nil {
		t.Fatalf("Failed to insert account: %v", err)
	}

	fetchedAcc, err := db.GetAccount(id)
	if err != nil {
		t.Fatalf("Failed to get account: %v", err)
	}

	if fetchedAcc.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", fetchedAcc.Email)
	}
	if fetchedAcc.DID != "did:aftersmtp:msgs.global:ryan" {
		t.Errorf("Expected DID match")
	}

	list, err := db.ListAccounts()
	if err != nil || len(list) != 1 {
		t.Fatalf("List accounts failed or returned incorrect length")
	}

	// 2. Folder Lookup (Schema initialization creates defaults)
	folderID, err := db.GetFolderByName("Inbox")
	if err != nil {
		t.Fatalf("Failed to get default Inbox folder: %v", err)
	}

	// 3. Message CRUD with Attachments
	msg := &accounts.Message{
		AccountID:   fetchedAcc.ID,
		FolderID:    int64(folderID),
		RemoteID:    "123-abc",
		Protocol:    "amp",
		Sender:      "alice@example.com",
		Recipients:  []string{"bob@example.com", "charlie@example.com"},
		Subject:     "Test Message",
		BodyPlain:   "Hello World",
		Flags:       []string{"\\Seen", "\\Flagged"},
		StakeAmount: 0.05,
		IPFSCID:     "QmTest123",
		Attachments: []accounts.Attachment{
			{Filename: "test.txt", ContentType: "text/plain", Size: 11, Data: []byte("Hello World")},
		},
	}

	msgID, err := db.SaveMessage(msg)
	if err != nil || msgID == 0 {
		t.Fatalf("Failed to save message: %v", err)
	}

	fetchedMsg, err := db.GetMessage(msgID)
	if err != nil {
		t.Fatalf("Failed to get message: %v", err)
	}

	if fetchedMsg.Subject != "Test Message" {
		t.Errorf("Expected Subject 'Test Message', got %s", fetchedMsg.Subject)
	}
	if len(fetchedMsg.Recipients) != 2 || fetchedMsg.Recipients[0] != "bob@example.com" {
		t.Errorf("Recipients array mismatch")
	}
	if len(fetchedMsg.Flags) != 2 || fetchedMsg.Flags[0] != "\\Seen" {
		t.Errorf("Flags array mismatch")
	}
	if len(fetchedMsg.Attachments) != 1 || fetchedMsg.Attachments[0].Filename != "test.txt" {
		t.Errorf("Attachments mismatch")
	}
	if string(fetchedMsg.Attachments[0].Data) != "Hello World" {
		t.Errorf("Attachment data mismatch")
	}
}
