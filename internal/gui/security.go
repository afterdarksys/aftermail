package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/security"
)

func buildSecurityTab() fyne.CanvasObject {
	domainEntry := widget.NewEntry()
	domainEntry.SetPlaceHolder("Domain to verify (e.g., example.com)")

	resultsBox := widget.NewMultiLineEntry()
	resultsBox.Disable()
	resultsBox.SetPlaceHolder("DNS Security check results will appear here...")
	resultsBox.Wrapping = fyne.TextWrapWord

	verifyBtn := widget.NewButton("Verify SPF, DMARC, MTA-STS, BIMI", func() {
		domain := domainEntry.Text
		if domain == "" {
			resultsBox.SetText("Please enter a domain to verify.")
			return
		}

		resultsBox.SetText(fmt.Sprintf("Querying DNS records for %s...\n\n", domain))

		spfRes := security.VerifySPF(domain)
		dmarcRes := security.VerifyDMARC(domain)
		stsRes := security.VerifyMTASTS(domain)
		bimiRes := security.VerifyBIMI(domain, "default")
		senderIDRes := security.VerifySenderID(domain) // Legacy check

		results := []security.CheckResult{spfRes, dmarcRes, stsRes, bimiRes, senderIDRes}
		
		output := resultsBox.Text
		for _, res := range results {
			status := "❌ FAIL"
			if res.Passed {
				status = "✅ PASS"
			}
			output += fmt.Sprintf("%s | [%s] - %s\n", status, res.Protocol, res.Message)
		}
		
		resultsBox.SetText(output)
	})

	form := container.NewHBox(
		widget.NewLabel("Domain:"),
		domainEntry,
		verifyBtn,
	)

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Domain DNS Security Checks", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			form,
		),
		nil, nil, nil,
		resultsBox,
	)
}
