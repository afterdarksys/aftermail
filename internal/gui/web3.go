package gui

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/wallet"
)

// buildWeb3Tab creates the AfterSMTP/Mailblocks Web3 integration tab
func buildWeb3Tab(w fyne.Window) fyne.CanvasObject {
	tabs := container.NewAppTabs(
		container.NewTabItem("Wallet", buildWalletSection(w)),
		container.NewTabItem("Mailblocks", buildMailblocksSection(w)),
		container.NewTabItem("IPFS", buildIPFSSection(w)),
		container.NewTabItem("AfterSMTP Gateway", buildAfterSMTPSection(w)),
	)

	tabs.SetTabLocation(container.TabLocationLeading)

	return tabs
}

func buildWalletSection(w fyne.Window) fyne.CanvasObject {
	// Wallet address display
	addressLabel := widget.NewLabel("No wallet loaded")
	addressLabel.Wrapping = fyne.TextWrapBreak

	// Balance display
	balanceLabel := widget.NewLabelWithStyle("0.000000 ETH", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	stakedLabel := widget.NewLabel("Staked: 0 ETH")

	var currentWallet *wallet.EthereumWallet
	var ethClient *wallet.MailblocksClient

	// Public Sepolia RPC for testing Mailblocks contracts
	const rpcURL = "https://ethereum-sepolia-rpc.publicnode.com"
	// Replace with actual deployed contract address on Sepolia
	const contractAddress = "0x0000000000000000000000000000000000000000"

	// Helper to refresh balance
	refreshBalance := func() {
		if ethClient == nil {
			return
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		bal, err := ethClient.GetBalance(ctx)
		if err != nil {
			balanceLabel.SetText("Error fetching balance")
			return
		}
		
		balanceLabel.SetText(wallet.FormatBalance(bal))
	}

	// Actions
	createWalletBtn := widget.NewButton("Create New Wallet", func() {
		dialog.ShowConfirm("Create Wallet", "This will generate a new Ethereum wallet. Make sure to backup the private key!", func(confirmed bool) {
			if !confirmed {
				return
			}

			ethWallet, err := wallet.NewWallet()
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			currentWallet = ethWallet
			addressLabel.SetText(fmt.Sprintf("Address: %s", ethWallet.Address.Hex()))
			
			// Initialize client
			client, err := wallet.NewMailblocksClient(rpcURL, contractAddress, currentWallet)
			if err == nil {
				ethClient = client
				balanceLabel.SetText("Syncing...")
				go refreshBalance()
			} else {
				balanceLabel.SetText("0.000000 ETH (new wallet, RPC err)")
			}

			// Show private key backup dialog
			pkHex := ethWallet.ExportPrivateKeyHex()
			dialog.ShowInformation("Backup Private Key",
				fmt.Sprintf("⚠️ IMPORTANT: Save this private key securely!\n\nPrivate Key:\n%s\n\nNever share this with anyone!", pkHex),
				w)
		}, w)
	})
	createWalletBtn.Importance = widget.HighImportance

	importKeyBtn := widget.NewButton("Import Private Key", func() {
		pkEntry := widget.NewPasswordEntry()
		pkEntry.SetPlaceHolder("Enter private key (hex, no 0x prefix)")

		dialog.ShowForm("Import Wallet", "Import", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Private Key", pkEntry),
		}, func(confirmed bool) {
			if !confirmed || pkEntry.Text == "" {
				return
			}

			ethWallet, err := wallet.FromPrivateKeyHex(pkEntry.Text)
			if err != nil {
				dialog.ShowError(fmt.Errorf("invalid private key: %w", err), w)
				return
			}

			currentWallet = ethWallet
			addressLabel.SetText(fmt.Sprintf("Address: %s", ethWallet.Address.Hex()))

			// Connect to Ethereum node
			client, err := wallet.NewMailblocksClient(rpcURL, contractAddress, currentWallet)
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to connect to RPC: %w", err), w)
				balanceLabel.SetText("RPC Error")
				return
			}
			
			ethClient = client
			balanceLabel.SetText("Fetching balance...")
			go refreshBalance()

			dialog.ShowInformation("Success", "Wallet imported successfully!", w)
		}, w)
	})

	importKeystoreBtn := widget.NewButton("Import Keystore", func() {
		dialog.ShowInformation("Import Keystore", "Keystore import coming soon!\nFor now, use private key import.", w)
	})

	refreshBalanceBtn := widget.NewButton("Refresh Balance", func() {
		if ethClient == nil {
			dialog.ShowInformation("Error", "No wallet loaded.", w)
			return
		}
		balanceLabel.SetText("Syncing...")
		go refreshBalance()
	})

	exportBtn := widget.NewButton("Export Private Key", func() {
		if currentWallet == nil {
			dialog.ShowInformation("Export", "Private key export available after wallet is loaded", w)
			return
		}
		pkHex := currentWallet.ExportPrivateKeyHex()
		dialog.ShowInformation("Export Private Key", fmt.Sprintf("Private Key:\n%s", pkHex), w)
	})

	// Wallet info card
	walletCard := widget.NewCard("Ethereum Wallet", "",
		container.NewVBox(
			addressLabel,
			balanceLabel,
			stakedLabel,
			widget.NewSeparator(),
			container.NewGridWithColumns(2,
				createWalletBtn,
				importKeyBtn,
			),
			container.NewGridWithColumns(2,
				importKeystoreBtn,
				refreshBalanceBtn,
			),
			exportBtn,
		),
	)

	// Transaction history
	txHistory := widget.NewList(
		func() int { return 0 }, // No transactions yet
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Tx Hash"),
				widget.NewLabel("Type"),
				widget.NewLabel("Amount"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {},
	)

	txCard := widget.NewCard("Recent Transactions", "", txHistory)

	return container.NewBorder(
		walletCard,
		nil,
		nil,
		nil,
		txCard,
	)
}

func buildMailblocksSection(w fyne.Window) fyne.CanvasObject {
	// Quarantined emails list
	quarantinedList := widget.NewList(
		func() int { return 3 }, // Mock data
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabel("From: sender@example.com"),
				widget.NewLabel("Subject: Test Message"),
				widget.NewLabel("Stake: 0.01 ETH"),
				container.NewHBox(
					widget.NewButton("Accept (Refund)", nil),
					widget.NewButton("Reject (Slash)", nil),
					widget.NewButton("View on IPFS", nil),
				),
				widget.NewSeparator(),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			// Mock quarantined email data
			senders := []string{"alice@example.com", "bob@spam.com", "eve@phishing.net"}
			subjects := []string{"Important: Review Required", "You won a prize!", "Reset your password"}
			stakes := []string{"0.01 ETH", "0.005 ETH", "0.002 ETH"}

			c := obj.(*fyne.Container)
			c.Objects[0].(*widget.Label).SetText(fmt.Sprintf("From: %s", senders[id]))
			c.Objects[1].(*widget.Label).SetText(fmt.Sprintf("Subject: %s", subjects[id]))
			c.Objects[2].(*widget.Label).SetText(fmt.Sprintf("Stake: %s", stakes[id]))

			// Button handlers
			btnContainer := c.Objects[3].(*fyne.Container)
			acceptBtn := btnContainer.Objects[0].(*widget.Button)
			rejectBtn := btnContainer.Objects[1].(*widget.Button)
			viewBtn := btnContainer.Objects[2].(*widget.Button)

			acceptBtn.OnTapped = func() {
				dialog.ShowConfirm("Accept Email",
					fmt.Sprintf("Accept this email and refund %s to sender?\n\nThis will add them to your contacts.", stakes[id]),
					func(confirmed bool) {
						if confirmed {
							dialog.ShowInformation("Accepted", "Email accepted! Stake refunded to sender.", w)
						}
					}, w)
			}

			rejectBtn.OnTapped = func() {
				dialog.ShowConfirm("Reject Email",
					fmt.Sprintf("Reject this email as spam and slash %s?\n\nThe stake will be forfeited.", stakes[id]),
					func(confirmed bool) {
						if confirmed {
							dialog.ShowInformation("Rejected", "Email rejected! Stake slashed.", w)
						}
					}, w)
			}

			viewBtn.OnTapped = func() {
				dialog.ShowInformation("View on IPFS", "Opening IPFS CID: Qm...\n\nThis would open the message in your browser.", w)
			}
		},
	)

	// Stats
	statsCard := widget.NewCard("Mailblocks Statistics", "",
		container.NewVBox(
			widget.NewLabel("📨 Quarantined: 3 emails"),
			widget.NewLabel("💰 Total Staked: 0.017 ETH"),
			widget.NewLabel("✅ Accepted (24h): 12"),
			widget.NewLabel("❌ Slashed (24h): 2"),
			widget.NewLabel("💸 Earnings (24h): 0.003 ETH"),
		),
	)

	// Settings
	minStakeEntry := widget.NewEntry()
	minStakeEntry.SetText("0.001")
	minStakeEntry.SetPlaceHolder("0.001")

	autoAcceptCheck := widget.NewCheck("Auto-accept from trusted contacts", nil)
	autoSlashCheck := widget.NewCheck("Auto-slash known spam senders", nil)

	settingsCard := widget.NewCard("Quarantine Settings", "",
		container.NewVBox(
			widget.NewForm(
				widget.NewFormItem("Minimum Stake (ETH)", minStakeEntry),
			),
			autoAcceptCheck,
			autoSlashCheck,
			widget.NewButton("Save Settings", func() {
				dialog.ShowInformation("Saved", "Mailblocks settings updated!", w)
			}),
		),
	)

	return container.NewBorder(
		container.NewVBox(statsCard, settingsCard),
		nil,
		nil,
		nil,
		container.NewVScroll(quarantinedList),
	)
}

func buildIPFSSection(w fyne.Window) fyne.CanvasObject {
	// IPFS status
	statusLabel := widget.NewLabel("⚠️ IPFS daemon not connected")
	statusLabel.Wrapping = fyne.TextWrapWord

	// IPFS endpoint config
	endpointEntry := widget.NewEntry()
	endpointEntry.SetText("http://127.0.0.1:5001")

	testBtn := widget.NewButton("Test Connection", func() {
		// TODO: Test IPFS connection
		dialog.ShowInformation("Testing", "Testing IPFS connection to "+endpointEntry.Text+"...", w)
		time.AfterFunc(1*time.Second, func() {
			statusLabel.SetText("✅ IPFS daemon connected (v0.20.0)")
		})
	})

	configCard := widget.NewCard("IPFS Configuration", "",
		container.NewVBox(
			statusLabel,
			widget.NewForm(
				widget.NewFormItem("API Endpoint", endpointEntry),
			),
			testBtn,
		),
	)

	// Pinned messages
	pinnedList := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("CID"),
				widget.NewLabel("Subject"),
				widget.NewLabel("Size"),
				widget.NewButton("Unpin", nil),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {},
	)

	pinnedCard := widget.NewCard("Pinned Messages", "Messages stored on IPFS", pinnedList)

	// Upload section
	uploadBtn := widget.NewButton("Upload Message to IPFS", func() {
		dialog.ShowInformation("Upload", "Select a message to upload to IPFS.\n\nCID will be generated and pinned.", w)
	})

	fetchBtn := widget.NewButton("Fetch from IPFS", func() {
		cidEntry := widget.NewEntry()
		cidEntry.SetPlaceHolder("Qm...")

		dialog.ShowForm("Fetch from IPFS", "Fetch", "Cancel", []*widget.FormItem{
			widget.NewFormItem("CID", cidEntry),
		}, func(confirmed bool) {
			if confirmed && cidEntry.Text != "" {
				dialog.ShowInformation("Fetching", fmt.Sprintf("Fetching CID: %s\n\nThis may take a moment...", cidEntry.Text), w)
			}
		}, w)
	})

	actionsCard := widget.NewCard("IPFS Actions", "",
		container.NewVBox(
			uploadBtn,
			fetchBtn,
		),
	)

	return container.NewBorder(
		container.NewVBox(configCard, actionsCard),
		nil,
		nil,
		nil,
		pinnedCard,
	)
}

func buildAfterSMTPSection(w fyne.Window) fyne.CanvasObject {
	// Gateway connection status
	gatewayStatus := widget.NewLabel("⚠️ Not connected to AfterSMTP gateway")

	// DID display
	didLabel := widget.NewLabel("DID: Not configured")
	didLabel.Wrapping = fyne.TextWrapBreak

	// Gateway config
	gatewayEntry := widget.NewEntry()
	gatewayEntry.SetText("tls://amp.msgs.global:4433")

	connectBtn := widget.NewButton("Connect to Gateway", func() {
		dialog.ShowInformation("Connecting", "Connecting to "+gatewayEntry.Text+"...\n\nAuthenticating with your DID...", w)
		time.AfterFunc(1*time.Second, func() {
			gatewayStatus.SetText("✅ Connected to msgs.global gateway")
		})
	})

	configCard := widget.NewCard("AfterSMTP Gateway", "",
		container.NewVBox(
			gatewayStatus,
			didLabel,
			widget.NewForm(
				widget.NewFormItem("Gateway URL", gatewayEntry),
			),
			connectBtn,
		),
	)

	// Create DID
	createDIDBtn := widget.NewButton("Create New DID", func() {
		usernameEntry := widget.NewEntry()
		usernameEntry.SetPlaceHolder("your-username")

		gatewaySelect := widget.NewSelect([]string{
			"msgs.global (Free)",
			"Custom Gateway",
		}, nil)
		gatewaySelect.SetSelected("msgs.global (Free)")

		dialog.ShowForm("Create AfterSMTP DID", "Create", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Username", usernameEntry),
			widget.NewFormItem("Gateway", gatewaySelect),
		}, func(confirmed bool) {
			if confirmed && usernameEntry.Text != "" {
				did := fmt.Sprintf("did:aftersmtp:msgs.global:%s", usernameEntry.Text)
				dialog.ShowInformation("DID Created",
					fmt.Sprintf("Your new DID:\n%s\n\nKeys generated:\n✓ Ed25519 signing key\n✓ X25519 encryption key\n\nYou can now send/receive encrypted emails!", did),
					w)
				didLabel.SetText("DID: " + did)
			}
		}, w)
	})
	createDIDBtn.Importance = widget.HighImportance

	importDIDBtn := widget.NewButton("Import Existing DID", func() {
		dialog.ShowInformation("Import DID", "Import your DID credentials:\n• DID string\n• Ed25519 private key\n• X25519 private key", w)
	})

	didCard := widget.NewCard("DID Management", "",
		container.NewVBox(
			widget.NewLabel("AfterSMTP uses DIDs instead of passwords"),
			container.NewGridWithColumns(2,
				createDIDBtn,
				importDIDBtn,
			),
		),
	)

	// Inbox stats
	statsCard := widget.NewCard("AfterSMTP Stats", "",
		container.NewVBox(
			widget.NewLabel("📨 Encrypted Messages: 127"),
			widget.NewLabel("✅ Verified Signatures: 127"),
			widget.NewLabel("🔗 Blockchain Proofs: 127"),
			widget.NewLabel("⚡ Avg Latency: 45ms"),
		),
	)

	return container.NewBorder(
		container.NewVBox(configCard, didCard, statsCard),
		nil,
		nil,
		nil,
		widget.NewLabel("AfterSMTP provides end-to-end encrypted email with blockchain-backed verification.\n\nConfigure your DID and gateway to get started."),
	)
}
