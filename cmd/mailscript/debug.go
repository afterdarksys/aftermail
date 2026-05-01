package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/afterdarksys/aftermail/pkg/rules"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Visually step through a MailScript rule",
	Long: `Interactive debugger for MailScript rules.

Step line-by-line through a script to understand what fires and why.
Inspect variable state, test different message contexts, set breakpoints.

Examples:
  # Debug a script interactively
  mailscript debug --script=filter.star

  # Debug with a pre-configured message
  mailscript debug --script=filter.star --from="boss@evil.example" --subject="Urgent wire transfer"

  # Trace mode: run to completion and print all actions
  mailscript debug --script=filter.star --trace
`,
	RunE: runDebug,
}

var (
	debugFrom    string
	debugTo      string
	debugSubject string
	debugBody    string
	debugTrace   bool
	debugSpam    float64
)

func init() {
	rootCmd.AddCommand(debugCmd)
	debugCmd.Flags().StringVar(&scriptPath, "script", "", "Path to MailScript file (required)")
	debugCmd.Flags().StringVar(&debugFrom, "from", "test@example.com", "Sender address")
	debugCmd.Flags().StringVar(&debugTo, "to", "you@example.com", "Recipient address")
	debugCmd.Flags().StringVar(&debugSubject, "subject", "Test Message", "Email subject")
	debugCmd.Flags().StringVar(&debugBody, "body", "This is a test message body.", "Email body")
	debugCmd.Flags().Float64Var(&debugSpam, "spam-score", 0.0, "Spam score (0.0–10.0)")
	debugCmd.Flags().BoolVar(&debugTrace, "trace", false, "Trace mode: run to completion and show all actions")
	debugCmd.MarkFlagRequired("script")
}

// debugSession holds the mutable state for one debugging session.
type debugSession struct {
	script      string
	ctx         *rules.MessageContext
	lines       []string
	breakpoints map[int]bool
}

func runDebug(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script: %w", err)
	}

	ctx := &rules.MessageContext{
		Headers: map[string]string{
			"From":    debugFrom,
			"To":      debugTo,
			"Subject": debugSubject,
		},
		Body:        debugBody,
		MimeType:    "text/plain",
		SpamScore:   debugSpam,
		VirusStatus: "clean",
		Actions:     []string{},
		LogEntries:  []string{},
	}

	session := &debugSession{
		script:      string(data),
		ctx:         ctx,
		lines:       strings.Split(string(data), "\n"),
		breakpoints: map[int]bool{},
	}

	if debugTrace {
		return session.runTrace()
	}

	return session.runInteractive()
}

// runTrace executes the entire script and prints a comprehensive trace.
func (s *debugSession) runTrace() error {
	fmt.Println("━━━ MailScript Trace ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Script: %s\n", scriptPath)
	fmt.Printf("From:   %s\n", s.ctx.Headers["From"])
	fmt.Printf("Subj:   %s\n", s.ctx.Headers["Subject"])
	fmt.Printf("Spam:   %.1f\n", s.ctx.SpamScore)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	if err := rules.ExecuteEngine(s.script, s.ctx); err != nil {
		fmt.Printf("Runtime error: %v\n", err)
	}

	fmt.Println("─── Actions ───────────────────────────────────────────")
	if len(s.ctx.Actions) == 0 {
		fmt.Println("  (none — script completed without a terminal action)")
	}
	for i, a := range s.ctx.Actions {
		fmt.Printf("  [%d] %s\n", i+1, a)
	}

	if len(s.ctx.ModifiedHeaders) > 0 {
		fmt.Println("\n─── Modified Headers ──────────────────────────────────")
		keys := make([]string, 0, len(s.ctx.ModifiedHeaders))
		for k := range s.ctx.ModifiedHeaders {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("  %s: %s\n", k, s.ctx.ModifiedHeaders[k])
		}
	}

	if len(s.ctx.LogEntries) > 0 {
		fmt.Println("\n─── Log Entries ───────────────────────────────────────")
		for _, l := range s.ctx.LogEntries {
			fmt.Printf("  [log] %s\n", l)
		}
	}

	fmt.Println("\n─── Final Context ─────────────────────────────────────")
	fmt.Printf("  SpamScore:   %.1f\n", s.ctx.SpamScore)
	fmt.Printf("  VirusStatus: %s\n", s.ctx.VirusStatus)
	fmt.Printf("  MimeType:    %s\n", s.ctx.MimeType)
	fmt.Printf("  BodySize:    %d bytes\n", s.ctx.BodySize)

	return nil
}

// runInteractive is the interactive step debugger.
func (s *debugSession) runInteractive() error {
	scanner := bufio.NewScanner(os.Stdin)

	printDebugBanner(s)
	printHelp()
	fmt.Println()

	s.printScript(-1) // Show full script at start

	fmt.Println()
	fmt.Println("Type 'run' to execute, 'context' to inspect, 'help' for commands.")
	fmt.Println()

	for {
		fmt.Print("debug> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := parts[0]
		cmdArgs := parts[1:]

		switch cmd {
		case "run", "r":
			s.runAndReport()

		case "context", "ctx", "c":
			s.printContext()

		case "set":
			s.handleSet(cmdArgs)

		case "headers", "h":
			s.printHeaders()

		case "body", "b":
			fmt.Printf("Body:\n%s\n", s.ctx.Body)

		case "script", "s":
			n := -1
			if len(cmdArgs) > 0 {
				n, _ = strconv.Atoi(cmdArgs[0])
			}
			s.printScript(n)

		case "break", "bp":
			if len(cmdArgs) == 0 {
				s.listBreakpoints()
			} else {
				n, err := strconv.Atoi(cmdArgs[0])
				if err != nil {
					fmt.Printf("Invalid line number: %s\n", cmdArgs[0])
					continue
				}
				s.toggleBreakpoint(n)
			}

		case "reset":
			s.ctx.Actions = nil
			s.ctx.LogEntries = nil
			s.ctx.ModifiedHeaders = nil
			fmt.Println("Context reset.")

		case "help", "?":
			printHelp()

		case "exit", "quit", "q":
			fmt.Println("Goodbye.")
			return nil

		default:
			// Try to execute the line as a MailScript snippet
			snippet := line
			testCtx := &rules.MessageContext{
				Headers:     s.ctx.Headers,
				Body:        s.ctx.Body,
				MimeType:    s.ctx.MimeType,
				SpamScore:   s.ctx.SpamScore,
				VirusStatus: s.ctx.VirusStatus,
			}
			if err := rules.ExecuteEngine(snippet, testCtx); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else if len(testCtx.Actions) > 0 {
				for _, a := range testCtx.Actions {
					fmt.Printf("→ %s\n", a)
				}
			}
		}
	}

	return nil
}

func (s *debugSession) runAndReport() {
	// Reset actions for a clean run
	s.ctx.Actions = nil
	s.ctx.LogEntries = nil
	s.ctx.ModifiedHeaders = nil

	fmt.Println("Running script...")

	if err := rules.ExecuteEngine(s.script, s.ctx); err != nil {
		fmt.Printf("Runtime error: %v\n\n", err)
		return
	}

	fmt.Println("\n── Result ───────────────────────────────────────────")
	if len(s.ctx.Actions) == 0 {
		fmt.Println("  No terminal action — message will be accepted by default")
	} else {
		for i, a := range s.ctx.Actions {
			fmt.Printf("  [%d] %s\n", i+1, a)
		}
	}

	if len(s.ctx.ModifiedHeaders) > 0 {
		fmt.Println("\nModified headers:")
		for k, v := range s.ctx.ModifiedHeaders {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	if len(s.ctx.LogEntries) > 0 {
		fmt.Println("\nLog entries:")
		for _, l := range s.ctx.LogEntries {
			fmt.Printf("  [log] %s\n", l)
		}
	}
	fmt.Println()
}

func (s *debugSession) handleSet(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: set <field> <value>")
		fmt.Println("Fields: from, to, subject, body, spam, virus, mime")
		return
	}
	field := strings.ToLower(args[0])
	value := strings.Join(args[1:], " ")

	switch field {
	case "from":
		s.ctx.Headers["From"] = value
		fmt.Printf("From → %s\n", value)
	case "to":
		s.ctx.Headers["To"] = value
		fmt.Printf("To → %s\n", value)
	case "subject":
		s.ctx.Headers["Subject"] = value
		fmt.Printf("Subject → %s\n", value)
	case "body":
		s.ctx.Body = value
		fmt.Printf("Body → %q\n", value)
	case "spam":
		score, err := strconv.ParseFloat(value, 64)
		if err != nil {
			fmt.Printf("Invalid score: %s\n", value)
			return
		}
		s.ctx.SpamScore = score
		fmt.Printf("SpamScore → %.1f\n", score)
	case "virus":
		s.ctx.VirusStatus = value
		fmt.Printf("VirusStatus → %s\n", value)
	case "mime":
		s.ctx.MimeType = value
		fmt.Printf("MimeType → %s\n", value)
	default:
		// Try setting as a header
		s.ctx.Headers[args[0]] = value
		fmt.Printf("Header %s → %s\n", args[0], value)
	}
}

func (s *debugSession) printContext() {
	fmt.Println("\n── Message Context ──────────────────────────────────")
	fmt.Printf("  From:        %s\n", s.ctx.Headers["From"])
	fmt.Printf("  To:          %s\n", s.ctx.Headers["To"])
	fmt.Printf("  Subject:     %s\n", s.ctx.Headers["Subject"])
	fmt.Printf("  SpamScore:   %.1f\n", s.ctx.SpamScore)
	fmt.Printf("  VirusStatus: %s\n", s.ctx.VirusStatus)
	fmt.Printf("  MimeType:    %s\n", s.ctx.MimeType)
	fmt.Printf("  BodySize:    %d chars\n", len(s.ctx.Body))

	if len(s.ctx.Actions) > 0 {
		fmt.Printf("\n  Actions so far: %s\n", strings.Join(s.ctx.Actions, ", "))
	}
	fmt.Println()
}

func (s *debugSession) printHeaders() {
	fmt.Println("\n── Headers ──────────────────────────────────────────")
	keys := make([]string, 0, len(s.ctx.Headers))
	for k := range s.ctx.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-20s %s\n", k+":", s.ctx.Headers[k])
	}
	fmt.Println()
}

func (s *debugSession) printScript(highlight int) {
	fmt.Println("\n── Script ───────────────────────────────────────────")
	for i, line := range s.lines {
		lineNum := i + 1
		bp := "  "
		if s.breakpoints[lineNum] {
			bp = "● "
		}
		marker := "  "
		if lineNum == highlight {
			marker = "→ "
		}
		fmt.Printf("%s%s%3d │ %s\n", bp, marker, lineNum, line)
	}
	fmt.Println()
}

func (s *debugSession) toggleBreakpoint(n int) {
	if n < 1 || n > len(s.lines) {
		fmt.Printf("Line %d is out of range (script has %d lines)\n", n, len(s.lines))
		return
	}
	if s.breakpoints[n] {
		delete(s.breakpoints, n)
		fmt.Printf("Breakpoint removed at line %d\n", n)
	} else {
		s.breakpoints[n] = true
		fmt.Printf("Breakpoint set at line %d: %s\n", n, strings.TrimSpace(s.lines[n-1]))
	}
}

func (s *debugSession) listBreakpoints() {
	if len(s.breakpoints) == 0 {
		fmt.Println("No breakpoints set.")
		return
	}
	bps := make([]int, 0, len(s.breakpoints))
	for n := range s.breakpoints {
		bps = append(bps, n)
	}
	sort.Ints(bps)
	fmt.Println("Breakpoints:")
	for _, n := range bps {
		fmt.Printf("  Line %d: %s\n", n, strings.TrimSpace(s.lines[n-1]))
	}
}

func printDebugBanner(s *debugSession) {
	fmt.Println("━━━ MailScript Debugger ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Script: %s (%d lines)\n", scriptPath, len(s.lines))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func printHelp() {
	fmt.Println(`Commands:
  run (r)            Execute script with current context, show actions
  context (ctx, c)   Show current message context
  set <field> <val>  Set context field (from/to/subject/body/spam/virus/mime)
                     Or set any header: set X-Priority 1
  headers (h)        List all current headers
  body (b)           Show message body
  script (s) [line]  Show script source (highlight optional line)
  break (bp) [line]  Toggle breakpoint at line (list all if no arg)
  reset              Reset actions/log for a fresh run
  help (?)           Show this help
  exit (q)           Exit debugger`)
}
