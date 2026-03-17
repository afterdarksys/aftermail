package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/plugins"
)

// SolidityEditorTab creates the Solidity smart contract editor interface
func SolidityEditorTab(w fyne.Window) fyne.CanvasObject {
	// Initialize Solidity plugin
	homeDir, _ := os.UserHomeDir()
	contractsDir := filepath.Join(homeDir, ".aftermail", "contracts")
	_ = os.MkdirAll(contractsDir, 0755)

	solPlugin, err := plugins.NewSolidityPlugin(contractsDir)
	if err != nil {
		// Show error if solc not installed
		return container.NewCenter(
			container.NewVBox(
				widget.NewLabel("⚠️ Solidity Compiler Not Found"),
				widget.NewLabel("Install solc to enable Solidity development:"),
				widget.NewLabel("npm install -g solc"),
				widget.NewLabel(""),
				widget.NewLabel("Or use system package manager:"),
				widget.NewLabel("brew install solidity (macOS)"),
				widget.NewLabel("apt install solc (Ubuntu)"),
			),
		)
	}

	// Code editor (multi-line entry as basic editor)
	codeEditor := widget.NewMultiLineEntry()
	codeEditor.SetPlaceHolder("// Write your Solidity smart contract here...\n// SPDX-License-Identifier: MIT\npragma solidity ^0.8.20;\n\ncontract MyContract {\n    // Add your code here\n}")
	codeEditor.Wrapping = fyne.TextWrapOff

	// File path entry
	filePathEntry := widget.NewEntry()
	filePathEntry.SetPlaceHolder("contract.sol")

	// Compilation output
	outputLog := widget.NewMultiLineEntry()
	outputLog.Disable()
	outputLog.SetPlaceHolder("Compilation output will appear here...")
	outputLog.Wrapping = fyne.TextWrapWord

	// New Contract button
	newContractBtn := widget.NewButton("New Contract", func() {
		templates := []string{"Basic", "ERC20 Token", "ERC721 NFT"}
		templateSelect := widget.NewSelect(templates, func(selected string) {})
		templateSelect.SetSelected("Basic")

		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("MyContract")

		formItems := []*widget.FormItem{
			widget.NewFormItem("Template", templateSelect),
			widget.NewFormItem("Name", nameEntry),
		}

		dialog.ShowForm("Create New Contract", "Create", "Cancel", formItems, func(confirmed bool) {
			if !confirmed || nameEntry.Text == "" {
				return
			}

			var templateType string
			switch templateSelect.Selected {
			case "Basic":
				templateType = "basic"
			case "ERC20 Token":
				templateType = "erc20"
			case "ERC721 NFT":
				templateType = "erc721"
			}

			filename, err := solPlugin.CreateTemplate(nameEntry.Text, templateType)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Failed to create contract: %w", err), w)
				return
			}

			// Load the created file
			content, _ := os.ReadFile(filename)
			codeEditor.SetText(string(content))
			filePathEntry.SetText(filepath.Base(filename))
			outputLog.SetText(fmt.Sprintf("✓ Created new %s contract: %s", templateSelect.Selected, filename))
		}, w)
	})

	// Load button
	loadBtn := widget.NewButton("Load File", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()

			content, err := os.ReadFile(reader.URI().Path())
			if err != nil {
				dialog.ShowError(fmt.Errorf("Failed to read file: %w", err), w)
				return
			}

			codeEditor.SetText(string(content))
			filePathEntry.SetText(filepath.Base(reader.URI().Path()))
			outputLog.SetText(fmt.Sprintf("✓ Loaded: %s", reader.URI().Path()))
		}, w)
	})

	// Save button
	saveBtn := widget.NewButton("Save", func() {
		if filePathEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("Please specify a filename"), w)
			return
		}

		filename := filepath.Join(contractsDir, filePathEntry.Text)
		if !strings.HasSuffix(filename, ".sol") {
			filename += ".sol"
		}

		err := os.WriteFile(filename, []byte(codeEditor.Text), 0644)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to save: %w", err), w)
			return
		}

		outputLog.SetText(fmt.Sprintf("✓ Saved: %s", filename))
	})

	// Validate Syntax button
	validateBtn := widget.NewButton("Validate Syntax", func() {
		if filePathEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("Please save the file first"), w)
			return
		}

		filename := filepath.Join(contractsDir, filePathEntry.Text)
		if !strings.HasSuffix(filename, ".sol") {
			filename += ".sol"
		}

		// Save current content first
		_ = os.WriteFile(filename, []byte(codeEditor.Text), 0644)

		errors, err := solPlugin.ValidateSyntax(filename)
		if err != nil {
			outputLog.SetText(fmt.Sprintf("❌ Validation error: %v", err))
			return
		}

		if len(errors) == 0 {
			outputLog.SetText("✓ Syntax is valid!")
		} else {
			outputLog.SetText(fmt.Sprintf("⚠️ Found %d issue(s):\n\n%s", len(errors), strings.Join(errors, "\n")))
		}
	})

	// Compile button
	compileBtn := widget.NewButton("Compile", func() {
		if filePathEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("Please save the file first"), w)
			return
		}

		filename := filepath.Join(contractsDir, filePathEntry.Text)
		if !strings.HasSuffix(filename, ".sol") {
			filename += ".sol"
		}

		// Save current content first
		_ = os.WriteFile(filename, []byte(codeEditor.Text), 0644)

		outputLog.SetText("Compiling...")

		result, err := solPlugin.CompileContract(filename)
		if err != nil {
			outputLog.SetText(fmt.Sprintf("❌ Compilation failed:\n\n%v", err))
			return
		}

		if result.Success {
			// Pretty print the output
			output := fmt.Sprintf("✓ Compilation successful!\n\n")
			output += fmt.Sprintf("Source: %s\n", result.SourceFile)
			output += fmt.Sprintf("\nCompiled contract data saved.\n")
			output += fmt.Sprintf("\nYou can now deploy this contract using the Web3 tab.")

			// Save ABI and bytecode for later use
			abiFile := strings.TrimSuffix(filename, ".sol") + ".abi"
			binFile := strings.TrimSuffix(filename, ".sol") + ".bin"

			// Extract from combined JSON (simplified)
			_ = os.WriteFile(abiFile, []byte(result.Output), 0644)
			_ = os.WriteFile(binFile, []byte(result.Output), 0644)

			outputLog.SetText(output)
		} else {
			outputLog.SetText(fmt.Sprintf("❌ Compilation failed:\n\n%s", result.Output))
		}
	})

	// Format button
	formatBtn := widget.NewButton("Format Code", func() {
		if filePathEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("Please save the file first"), w)
			return
		}

		filename := filepath.Join(contractsDir, filePathEntry.Text)
		if !strings.HasSuffix(filename, ".sol") {
			filename += ".sol"
		}

		// Save current content first
		_ = os.WriteFile(filename, []byte(codeEditor.Text), 0644)

		err := solPlugin.FormatCode(filename)
		if err != nil {
			outputLog.SetText(fmt.Sprintf("⚠️ Formatting skipped (prettier not installed): %v\n\nInstall with: npm install -g prettier prettier-plugin-solidity", err))
			return
		}

		// Reload formatted content
		content, _ := os.ReadFile(filename)
		codeEditor.SetText(string(content))
		outputLog.SetText("✓ Code formatted successfully!")
	})

	// Help button
	helpBtn := widget.NewButton("Help", func() {
		helpText := `Solidity Smart Contract Editor

Features:
• Create new contracts from templates (Basic, ERC20, ERC721)
• Load and save Solidity files
• Syntax validation
• Compilation with optimization
• Code formatting (requires prettier)

Keyboard Shortcuts:
• Ctrl+S / Cmd+S: Save (use regular save button)

Requirements:
• solc: Solidity compiler (npm install -g solc)
• prettier: Code formatter (npm install -g prettier prettier-plugin-solidity)

Workflow:
1. Create a new contract or load existing one
2. Write/edit your Solidity code
3. Validate syntax
4. Compile the contract
5. Deploy using Web3 tab (if available)

Tips:
• Always include SPDX license identifier
• Use pragma solidity ^0.8.20 or newer
• Test thoroughly before deploying to mainnet
• Use OpenZeppelin libraries for standard contracts
`
		dialog.ShowInformation("Solidity Editor Help", helpText, w)
	})

	// Top toolbar
	toolbar := container.NewHBox(
		newContractBtn,
		loadBtn,
		saveBtn,
		validateBtn,
		compileBtn,
		formatBtn,
		helpBtn,
	)

	// File info bar
	fileBar := container.NewBorder(
		nil, nil,
		widget.NewLabel("File:"),
		nil,
		filePathEntry,
	)

	// Editor area
	editorArea := container.NewBorder(
		fileBar,
		nil,
		nil,
		nil,
		codeEditor,
	)

	// Split view: editor on left, output on right
	splitContent := container.NewHSplit(
		editorArea,
		container.NewBorder(
			widget.NewLabel("Output"),
			nil,
			nil,
			nil,
			outputLog,
		),
	)
	splitContent.SetOffset(0.6) // 60% editor, 40% output

	// Main layout
	return container.NewBorder(
		toolbar, // Top
		nil,     // Bottom
		nil,     // Left
		nil,     // Right
		splitContent,
	)
}
