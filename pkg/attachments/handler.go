package attachments

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/afterdarksys/aftermail/pkg/accounts"
)

// Handler manages pushing and pulling complex binary attachments 
// between the local AfterMail client constraints and remote hosts
type Handler struct {
	BaseDir string
}

// NewHandler scaffolds the default attachment engine mapped to `~/.aftermail/attachments/`
func NewHandler(baseDir string) *Handler {
	_ = os.MkdirAll(baseDir, 0700)
	return &Handler{
		BaseDir: baseDir,
	}
}

// Download extracts a payload block from memory or remote dial stream into a designated secure disk chunk
func (h *Handler) Download(att *accounts.Attachment, destFile string) error {
	if len(att.Data) == 0 {
		return fmt.Errorf("attachment buffer empty during fetch attempt")
	}

	// Verify Hash Integrity locally before executing flush
	hash := sha256.Sum256(att.Data)
	computedRef := hex.EncodeToString(hash[:])
	
	if att.Hash != "" && computedRef != att.Hash {
		log.Printf("[Warning] Attachment Integrity Mismatch: Expected %s, Got %s", att.Hash, computedRef)
		return fmt.Errorf("attachment hash signature invalid, dropping payload")
	}

	outPath := filepath.Join(h.BaseDir, destFile)
	if err := os.WriteFile(outPath, att.Data, 0600); err != nil {
		return fmt.Errorf("failed to flush buffer to disk constraints: %w", err)
	}

	log.Printf("[Attachments] Successfully downloaded: %s (%d bytes)", destFile, len(att.Data))
	return nil
}

// Upload converts a local disk structure into an AMF/MIME compatible Attachment Object
func (h *Handler) Upload(sourceFile string) (*accounts.Attachment, error) {
	file, err := os.Open(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("cannot stat file for upload mapping: %w", err)
	}
	defer file.Close()

	stat, _ := file.Stat()
	if stat.Size() > 100*1024*1024 { // 100MB UI sanity limit per object inside pure RAM payloads
		return nil, fmt.Errorf("requested attachment %s exceeds 100MB static heap chunk limit", stat.Name())
	}

	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file bytes: %w", err)
	}

	hash := sha256.Sum256(raw)
	
	// Create struct descriptor
	att := &accounts.Attachment{
		Filename:    stat.Name(),
		ContentType: "application/octet-stream", // generic fallback, ideally we sniff with http.DetectContentType(raw[:512])
		Size:        stat.Size(),
		Data:        raw,
		Hash:        hex.EncodeToString(hash[:]),
	}
	
	log.Printf("[Attachments] Successfully staged: %s (%d bytes)", att.Filename, att.Size)

	return att, nil
}
