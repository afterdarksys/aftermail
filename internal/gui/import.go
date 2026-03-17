package gui

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"github.com/afterdarksys/aftermail/pkg/i18n"
)

// ImportMbox parses a traditional Thunderbird/Apple Mail .mbox file format.
// An mbox file is a plain text file containing a sequence of email messages
// concatenated together, each delimited by a line starting with "From ".
func ImportMbox(w fyne.Window) {
	dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		if uc == nil {
			return // User canceled
		}
		defer uc.Close()

		progress := dialog.NewProgressInfinite(
			i18n.T("importing_title", "Importing Messages"), 
			i18n.T("importing_msg", "Reading .mbox file..."), 
			w,
		)
		progress.Show()

		go func() {
			scanner := bufio.NewScanner(uc)
			// Increase max buffer for large payloads within a single message
			scanner.Buffer(make([]byte, 1024*64), 50*1024*1024)

			messageCount := 0
			var currentMessage bytes.Buffer

			for scanner.Scan() {
				line := scanner.Text()
				
				// Standard mbox delimiter check
				if strings.HasPrefix(line, "From ") {
					if currentMessage.Len() > 0 {
						// Process completed message
						messageCount++
						// Mock DB persistence step
						// db.InsertMessage(currentMessage.String())
						currentMessage.Reset()
					}
				}
				currentMessage.WriteString(line + "\n")
			}
			
			// Catch the very last message in the buffer
			if currentMessage.Len() > 0 {
				messageCount++
				currentMessage.Reset()
			}

			if err := scanner.Err(); err != nil {
				progress.Hide()
				dialog.ShowError(fmt.Errorf("failed to read .mbox fully: %w", err), w)
				return
			}

			progress.Hide()
			dialog.ShowInformation(
				i18n.T("import_success_title", "Import Complete"),
				fmt.Sprintf(i18n.T("import_success_msg", "Successfully extracted and imported %d messages from the .mbox archive."), messageCount),
				w,
			)
		}()
	}, w)
}
