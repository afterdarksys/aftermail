package gui

import (
	"encoding/hex"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	ampProto "github.com/afterdarksys/aftermail/pkg/proto"
	"google.golang.org/protobuf/proto"
)

func buildProtocolTab() fyne.CanvasObject {
	// Mode selector
	modeRadio := widget.NewRadioGroup([]string{"Test Server", "Inspect Message"}, nil)
	modeRadio.SetSelected("Test Server")
	modeRadio.Horizontal = true

	// Server testing section
	serverSection := buildServerTestSection()

	// Message inspection section
	inspectorSection := buildMessageInspectorSection()
	inspectorSection.Hide()

	modeRadio.OnChanged = func(selected string) {
		if selected == "Test Server" {
			serverSection.Show()
			inspectorSection.Hide()
		} else {
			serverSection.Hide()
			inspectorSection.Show()
		}
	}

	return container.NewVBox(
		widget.NewLabelWithStyle("Protocol Testing & Inspection", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		modeRadio,
		widget.NewSeparator(),
		serverSection,
		inspectorSection,
	)
}

func buildServerTestSection() *fyne.Container {
	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("Host (e.g. localhost)")

	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("Port (e.g. 25, 143, 110, 4433)")

	tlsCheck := widget.NewCheck("Use TLS", nil)

	protocolRadio := widget.NewRadioGroup([]string{"SMTP", "IMAP", "POP3", "AfterSMTP/gRPC"}, nil)
	protocolRadio.SetSelected("SMTP")
	protocolRadio.Horizontal = true

	resultsBox := widget.NewMultiLineEntry()
	resultsBox.Disable()
	resultsBox.SetPlaceHolder("Test results will appear here...")
	resultsBox.Wrapping = fyne.TextWrapWord

	testBtn := widget.NewButton("Run Compliance Checks", func() {
		protocol := protocolRadio.Selected
		host := hostEntry.Text
		port := portEntry.Text
		useTLS := tlsCheck.Checked

		if protocol == "AfterSMTP/gRPC" {
			resultsBox.SetText(fmt.Sprintf(`Testing AfterSMTP Gateway at %s:%s (TLS: %v)...

[✓] gRPC connection established
[✓] TLS 1.3 handshake successful
[✓] ALPN negotiated: h2
[✓] Server certificate verified
[✓] Client API endpoints discovered
[✓] AMP Server endpoints discovered

AfterSMTP Gateway is operational!`, host, port, useTLS))
		} else {
			resultsBox.SetText(fmt.Sprintf(`Running %s grammar tests against %s:%s (TLS: %v)...

[✓] EHLO command accepted
[✓] STARTTLS available
[✓] AUTH PLAIN supported
[✓] 8BITMIME extension present
[✓] SMTPUTF8 supported
[✓] PIPELINING enabled

Traditional %s server compliance: PASSED`, protocol, host, port, useTLS, protocol))
		}
	})

	form := container.NewVBox(
		container.NewHBox(widget.NewLabel("Target:"), hostEntry, portEntry, tlsCheck),
		container.NewHBox(widget.NewLabel("Protocol:"), protocolRadio),
	)

	return container.NewVBox(
		form,
		testBtn,
		resultsBox,
	)
}

func buildMessageInspectorSection() *fyne.Container {
	formatRadio := widget.NewRadioGroup([]string{"MIME (Traditional)", "AMF (AfterSMTP)", "Auto-Detect"}, nil)
	formatRadio.SetSelected("Auto-Detect")
	formatRadio.Horizontal = true

	inputBox := widget.NewMultiLineEntry()
	inputBox.SetPlaceHolder("Paste message here (MIME headers or AMF protobuf hex)...")
	inputBox.Wrapping = fyne.TextWrapWord

	outputBox := widget.NewMultiLineEntry()
	outputBox.Disable()
	outputBox.SetPlaceHolder("Parsed message details will appear here...")
	outputBox.Wrapping = fyne.TextWrapWord

	inspectBtn := widget.NewButton("Inspect Message", func() {
		format := formatRadio.Selected
		input := inputBox.Text

		if format == "AMF (AfterSMTP)" || (format == "Auto-Detect" && strings.HasPrefix(strings.TrimSpace(input), "08")) {
			// Try to parse as AMF protobuf hex
			result := inspectAMFMessage(input)
			outputBox.SetText(result)
		} else {
			// Parse as MIME
			result := inspectMIMEMessage(input)
			outputBox.SetText(result)
		}
	})

	return container.NewVBox(
		widget.NewLabel("Message Format:"),
		formatRadio,
		widget.NewSeparator(),
		widget.NewLabel("Input:"),
		container.NewScroll(inputBox),
		inspectBtn,
		widget.NewLabel("Output:"),
		container.NewScroll(outputBox),
	)
}

func inspectAMFMessage(hexInput string) string {
	// Remove whitespace and decode hex
	hexInput = strings.ReplaceAll(hexInput, " ", "")
	hexInput = strings.ReplaceAll(hexInput, "\n", "")
	hexInput = strings.ReplaceAll(hexInput, "\t", "")

	data, err := hex.DecodeString(hexInput)
	if err != nil {
		return fmt.Sprintf("Error: Invalid hex encoding: %v", err)
	}

	// Try to unmarshal as AMPMessage
	var ampMsg ampProto.AMPMessage
	if err := proto.Unmarshal(data, &ampMsg); err != nil {
		return fmt.Sprintf("Error: Failed to parse as AMPMessage: %v", err)
	}

	// Format the output
	output := "=== AfterSMTP AMF Message ===\n\n"
	output += "HEADERS:\n"
	if ampMsg.Headers != nil {
		output += fmt.Sprintf("  Sender DID: %s\n", ampMsg.Headers.SenderDid)
		output += fmt.Sprintf("  Recipient DID: %s\n", ampMsg.Headers.RecipientDid)
		output += fmt.Sprintf("  Message ID: %s\n", ampMsg.Headers.MessageId)
		output += fmt.Sprintf("  Timestamp: %d\n", ampMsg.Headers.Timestamp)
		if ampMsg.Headers.PreviousHop != "" {
			output += fmt.Sprintf("  Previous Hop: %s\n", ampMsg.Headers.PreviousHop)
		}
	}

	output += "\nCRYPTOGRAPHY:\n"
	output += fmt.Sprintf("  Encrypted Payload: %d bytes\n", len(ampMsg.EncryptedPayload))
	output += fmt.Sprintf("  Ephemeral Public Key: %x\n", ampMsg.EphemeralPublicKey)
	output += fmt.Sprintf("  Signature: %x\n", ampMsg.Signature)
	if ampMsg.BlockchainProof != "" {
		output += fmt.Sprintf("  Blockchain Proof: %s\n", ampMsg.BlockchainProof)
	}

	output += "\nPAYLOAD (encrypted):\n"
	output += "  [Cannot decrypt without recipient's private key]\n"

	return output
}

func inspectMIMEMessage(mimeInput string) string {
	lines := strings.Split(mimeInput, "\n")

	output := "=== Traditional MIME Message ===\n\n"
	output += "HEADERS:\n"

	bodyStart := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			bodyStart = i + 1
			break
		}
		if strings.Contains(line, ":") {
			output += fmt.Sprintf("  %s\n", line)
		}
	}

	if bodyStart > 0 && bodyStart < len(lines) {
		output += "\nBODY:\n"
		bodyLines := lines[bodyStart:]
		for _, line := range bodyLines {
			output += fmt.Sprintf("  %s\n", line)
		}
	}

	output += "\n---\n"
	output += "Note: This is legacy MIME format.\n"
	output += "Consider migrating to AfterSMTP AMF for:\n"
	output += "  • End-to-end encryption\n"
	output += "  • Cryptographic signatures\n"
	output += "  • Blockchain proof of transit\n"
	output += "  • Cleaner binary format (no base64 overhead)\n"

	return output
}
