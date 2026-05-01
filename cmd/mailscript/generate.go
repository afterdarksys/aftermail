package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/afterdarksys/aftermail/pkg/ai"
	"github.com/afterdarksys/aftermail/pkg/rules"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a MailScript rule from a natural language description",
	Long: `Use AI to generate a MailScript (Starlark) rule from plain English.

Describe what you want your email filter to do — the AI will produce
a ready-to-use MailScript rule.

Examples:
  # Quick one-shot generation
  mailscript generate --prompt="Block any email mentioning invoice or payment from senders I've never emailed"

  # Interactive mode (enter description interactively)
  mailscript generate

  # Generate and immediately test against a message
  mailscript generate --prompt="File newsletters into a Newsletters folder" --test

  # Save to a file
  mailscript generate --prompt="Quarantine attachments over 10MB" --output=large-attach.star

  # Refine an existing script
  mailscript generate --refine=existing.star --prompt="Also add an exception for my boss"

  # Explain what a script does
  mailscript generate --explain=filter.star
`,
	RunE: runGenerate,
}

var (
	generatePrompt  string
	generateOutput  string
	generateRefine  string
	generateExplain string
	generateTest    bool
)

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVar(&generatePrompt, "prompt", "", "Natural language description of the rule")
	generateCmd.Flags().StringVar(&generateOutput, "output", "", "Save generated script to file")
	generateCmd.Flags().StringVar(&generateRefine, "refine", "", "Existing script file to refine")
	generateCmd.Flags().StringVar(&generateExplain, "explain", "", "Explain what a script does (plain English)")
	generateCmd.Flags().BoolVar(&generateTest, "test", false, "Test the generated script interactively after generation")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("set ANTHROPIC_API_KEY or OPENROUTER_API_KEY to use AI generation")
		}
	}

	provider := ai.ProviderAnthropic
	if os.Getenv("OPENROUTER_API_KEY") != "" && os.Getenv("ANTHROPIC_API_KEY") == "" {
		provider = ai.ProviderOpenRouter
	}

	assistant := ai.NewAssistant(provider, apiKey, "")
	ctx := context.Background()

	// --- Explain mode ---
	if generateExplain != "" {
		return runExplain(ctx, assistant, generateExplain)
	}

	// --- Refine mode ---
	if generateRefine != "" {
		return runRefine(ctx, assistant, generateRefine)
	}

	// --- Generate mode ---
	prompt := generatePrompt
	if prompt == "" {
		// Interactive prompt collection
		fmt.Println("Describe what you want your email filter to do:")
		fmt.Print("> ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			prompt = strings.TrimSpace(scanner.Text())
		}
	}

	if prompt == "" {
		return fmt.Errorf("no description provided")
	}

	fmt.Printf("\nGenerating MailScript rule for: %q\n\n", prompt)

	script, err := assistant.GenerateMailScript(ctx, prompt)
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	// Print the generated script
	fmt.Println("─── Generated MailScript ──────────────────────────────")
	fmt.Println(script)
	fmt.Println("───────────────────────────────────────────────────────")

	// Save to file if requested
	if generateOutput != "" {
		if err := os.WriteFile(generateOutput, []byte(script), 0644); err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}
		fmt.Printf("\nSaved to %s\n", generateOutput)
	}

	// Test mode: load the script and run through the REPL
	if generateTest {
		fmt.Println("\nRunning test mode — enter message parameters to test the rule.")
		ctx := &rules.MessageContext{
			Headers: map[string]string{
				"From":    "test@example.com",
				"To":      "you@example.com",
				"Subject": "Test Message",
			},
			Body:        "This is a test message body.",
			MimeType:    "text/plain",
			SpamScore:   0.0,
			VirusStatus: "clean",
		}
		if err := rules.ExecuteEngine(script, ctx); err != nil {
			fmt.Printf("Script error: %v\n", err)
		} else {
			printTestResult(ctx)
		}
	}

	return nil
}

func runExplain(ctx context.Context, a *ai.Assistant, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	fmt.Printf("Explaining: %s\n\n", filepath.Base(path))
	explanation, err := a.ExplainMailScript(ctx, string(data))
	if err != nil {
		return fmt.Errorf("explanation failed: %w", err)
	}

	fmt.Println(explanation)
	return nil
}

func runRefine(ctx context.Context, a *ai.Assistant, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	prompt := generatePrompt
	if prompt == "" {
		fmt.Printf("Loaded %s\nDescribe what you'd like to change or add:\n> ", filepath.Base(path))
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			prompt = strings.TrimSpace(scanner.Text())
		}
	}

	if prompt == "" {
		return fmt.Errorf("no change description provided")
	}

	fmt.Printf("\nRefining script with: %q\n\n", prompt)
	refined, err := a.RefineMailScript(ctx, string(data), prompt)
	if err != nil {
		return fmt.Errorf("refinement failed: %w", err)
	}

	fmt.Println("─── Refined MailScript ────────────────────────────────")
	fmt.Println(refined)
	fmt.Println("───────────────────────────────────────────────────────")

	if generateOutput != "" {
		if err := os.WriteFile(generateOutput, []byte(refined), 0644); err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}
		fmt.Printf("\nSaved to %s\n", generateOutput)
	}
	return nil
}

func printTestResult(ctx *rules.MessageContext) {
	fmt.Println("\nTest Result:")
	if len(ctx.Actions) == 0 {
		fmt.Println("  No actions taken (script fell through without a terminal action)")
	} else {
		for _, a := range ctx.Actions {
			fmt.Printf("  → %s\n", a)
		}
	}
	if len(ctx.ModifiedHeaders) > 0 {
		fmt.Println("  Modified headers:")
		for k, v := range ctx.ModifiedHeaders {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}
	if len(ctx.LogEntries) > 0 {
		fmt.Println("  Log entries:")
		for _, l := range ctx.LogEntries {
			fmt.Printf("    [log] %s\n", l)
		}
	}
}
