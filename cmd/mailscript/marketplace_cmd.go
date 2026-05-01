package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/afterdarksys/aftermail/pkg/marketplace"
	"github.com/spf13/cobra"
)

var marketplaceCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Browse, install, and publish MailScript rules",
	Long: `The MailScript Marketplace is a curated registry of ready-to-use
email filtering rules. Browse community scripts, install with one command,
and publish your own.

Examples:
  mailscript marketplace list             # List installed scripts
  mailscript marketplace browse           # Browse all available scripts
  mailscript marketplace search spam      # Search by keyword
  mailscript marketplace install invoice  # Install by name/ID
  mailscript marketplace builtins         # Install all built-in starter scripts
  mailscript marketplace export my.star   # Package a script for sharing
  mailscript marketplace import pkg.json  # Install from a shared JSON package
  mailscript marketplace remove invoice   # Remove an installed script
`,
}

var (
	marketplaceDir string
)

func init() {
	homeDir, _ := os.UserHomeDir()
	defaultDir := filepath.Join(homeDir, ".aftermail", "marketplace")

	rootCmd.AddCommand(marketplaceCmd)
	marketplaceCmd.PersistentFlags().StringVar(&marketplaceDir, "dir", defaultDir, "Marketplace registry directory")

	marketplaceCmd.AddCommand(marketplaceListCmd)
	marketplaceCmd.AddCommand(marketplaceBrowseCmd)
	marketplaceCmd.AddCommand(marketplaceSearchCmd)
	marketplaceCmd.AddCommand(marketplaceInstallCmd)
	marketplaceCmd.AddCommand(marketplaceBuiltinsCmd)
	marketplaceCmd.AddCommand(marketplaceExportCmd)
	marketplaceCmd.AddCommand(marketplaceImportCmd)
	marketplaceCmd.AddCommand(marketplaceRemoveCmd)
}

// --- list ---

var marketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed scripts",
	RunE: func(cmd *cobra.Command, args []string) error {
		reg, err := marketplace.NewRegistry(marketplaceDir)
		if err != nil {
			return err
		}

		scripts := reg.Installed()
		if len(scripts) == 0 {
			fmt.Println("No scripts installed. Run `marketplace builtins` to get started.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tVERSION\tTAGS")
		fmt.Fprintln(w, "──\t────\t───────\t────")
		for _, s := range scripts {
			fmt.Fprintf(w, "%s\t%s\tv%s\t%s\n",
				s.ID, s.Name, s.Version, strings.Join(s.Tags, ", "))
		}
		return w.Flush()
	},
}

// --- browse ---

var marketplaceBrowseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Browse all available built-in scripts",
	RunE: func(cmd *cobra.Command, args []string) error {
		scripts := marketplace.BuiltinScripts()

		reg, _ := marketplace.NewRegistry(marketplaceDir)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tINSTALLED")
		fmt.Fprintln(w, "──\t────\t───────────\t─────────")
		for _, s := range scripts {
			_, installed := reg.Get(s.ID)
			installedMark := ""
			if installed {
				installedMark = "✓"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				s.ID, s.Name, s.Description, installedMark)
		}
		return w.Flush()
	},
}

// --- search ---

var marketplaceSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search installed scripts by keyword",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reg, err := marketplace.NewRegistry(marketplaceDir)
		if err != nil {
			return err
		}

		results := reg.Search(args[0])
		if len(results) == 0 {
			fmt.Printf("No installed scripts match %q.\n", args[0])
			return nil
		}

		for _, s := range results {
			fmt.Printf("%-40s %s\n", s.ID, s.Description)
		}
		return nil
	},
}

// --- install ---

var marketplaceInstallCmd = &cobra.Command{
	Use:   "install <id>",
	Short: "Install a built-in script by ID or partial name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.ToLower(args[0])
		builtins := marketplace.BuiltinScripts()

		// Find matching builtin
		var match *marketplace.Script
		for _, s := range builtins {
			if strings.Contains(strings.ToLower(s.ID), query) ||
				strings.Contains(strings.ToLower(s.Name), query) {
				match = s
				break
			}
		}

		if match == nil {
			return fmt.Errorf("no built-in script found matching %q", args[0])
		}

		reg, err := marketplace.NewRegistry(marketplaceDir)
		if err != nil {
			return err
		}

		if err := reg.Install(match); err != nil {
			return err
		}

		fmt.Printf("Installed: %s (%s)\n", match.Name, match.ID)
		fmt.Printf("  %s\n", match.Description)

		// Also write the .star file for easy use
		starPath := filepath.Join(marketplaceDir, strings.ReplaceAll(match.ID, "/", "__")+".star")
		if err := os.WriteFile(starPath, []byte(match.Code), 0644); err == nil {
			fmt.Printf("  Script file: %s\n", starPath)
		}

		return nil
	},
}

// --- builtins ---

var marketplaceBuiltinsCmd = &cobra.Command{
	Use:   "builtins",
	Short: "Install all built-in starter scripts",
	RunE: func(cmd *cobra.Command, args []string) error {
		reg, err := marketplace.NewRegistry(marketplaceDir)
		if err != nil {
			return err
		}

		for _, s := range marketplace.BuiltinScripts() {
			if err := reg.Install(s); err != nil {
				fmt.Printf("Warning: failed to install %s: %v\n", s.ID, err)
				continue
			}
			starPath := filepath.Join(marketplaceDir, strings.ReplaceAll(s.ID, "/", "__")+".star")
			_ = os.WriteFile(starPath, []byte(s.Code), 0644)
			fmt.Printf("✓ %s\n", s.Name)
		}

		fmt.Printf("\nScripts installed to: %s\n", marketplaceDir)
		return nil
	},
}

// --- export ---

var marketplaceExportCmd = &cobra.Command{
	Use:   "export <script.star>",
	Short: "Package a .star script as a shareable JSON file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", args[0], err)
		}

		base := filepath.Base(args[0])
		name := strings.TrimSuffix(base, filepath.Ext(base))

		fmt.Printf("Script name (default: %q): ", name)
		var inputName string
		fmt.Scanln(&inputName)
		if strings.TrimSpace(inputName) != "" {
			name = strings.TrimSpace(inputName)
		}

		fmt.Print("Author (email or handle): ")
		var author string
		fmt.Scanln(&author)

		fmt.Print("Description (one sentence): ")
		var description string
		fmt.Scanln(&description)

		fmt.Print("Tags (comma-separated): ")
		var tagsInput string
		fmt.Scanln(&tagsInput)
		var tags []string
		for _, t := range strings.Split(tagsInput, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}

		s, err := marketplace.Publish(name, author, description, tags, string(data))
		if err != nil {
			return err
		}

		jsonData, err := marketplace.ExportJSON(s)
		if err != nil {
			return err
		}

		outPath := name + ".mailscript.json"
		if err := os.WriteFile(outPath, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", outPath, err)
		}

		fmt.Printf("\nPackaged as: %s\n", outPath)
		fmt.Printf("Script ID:   %s\n", s.ID)
		fmt.Printf("Hash:        %s\n", s.CodeHash)
		return nil
	},
}

// --- import ---

var marketplaceImportCmd = &cobra.Command{
	Use:   "import <package.json>",
	Short: "Install a script from a shared JSON package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", args[0], err)
		}

		// Preview before installing
		var preview struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Author      string `json:"author"`
			Description string `json:"description"`
		}
		_ = json.Unmarshal(data, &preview)
		fmt.Printf("Installing: %s by %s\n", preview.Name, preview.Author)
		fmt.Printf("  %s\n", preview.Description)

		reg, err := marketplace.NewRegistry(marketplaceDir)
		if err != nil {
			return err
		}

		s, err := reg.ImportJSON(data)
		if err != nil {
			return fmt.Errorf("install failed: %w", err)
		}

		fmt.Printf("Installed: %s\n", s.ID)
		return nil
	},
}

// --- remove ---

var marketplaceRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove an installed script",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reg, err := marketplace.NewRegistry(marketplaceDir)
		if err != nil {
			return err
		}

		if err := reg.Uninstall(args[0]); err != nil {
			return err
		}

		fmt.Printf("Removed: %s\n", args[0])
		return nil
	},
}
