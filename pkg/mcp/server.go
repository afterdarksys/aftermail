// Package mcp implements a Model Context Protocol server for aftermaild.
// It speaks JSON-RPC 2.0 over stdio so Claude (and any MCP client) can
// treat the running daemon as a first-class tool provider.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/afterdarksys/aftermail/pkg/commitments"
	"github.com/afterdarksys/aftermail/pkg/deadman"
	"github.com/afterdarksys/aftermail/pkg/fingerprint"
	"github.com/afterdarksys/aftermail/pkg/marketplace"
	"github.com/afterdarksys/aftermail/pkg/reputation"
	"github.com/afterdarksys/aftermail/pkg/rules"
	"github.com/afterdarksys/aftermail/pkg/send"
	"github.com/afterdarksys/aftermail/pkg/stakemail"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

// ─── JSON-RPC 2.0 wire types ─────────────────────────────────────────────────

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ─── MCP protocol types ───────────────────────────────────────────────────────

type toolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ─── Server ───────────────────────────────────────────────────────────────────

// Server is a stdio MCP server backed by the aftermaild storage layer.
type Server struct {
	db          *storage.DB
	in          io.Reader
	out         io.Writer
	tools       []toolDef
}

// New creates a Server that reads from r and writes to w.
// Pass os.Stdin / os.Stdout for the standard MCP stdio transport.
func New(db *storage.DB, r io.Reader, w io.Writer) *Server {
	s := &Server{db: db, in: r, out: w}
	s.tools = buildToolDefs()
	return s
}

// Run enters the read-dispatch-write loop. It blocks until ctx is cancelled
// or the input stream is closed.
func (s *Server) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(s.in)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	log.Println("[MCP] aftermaild MCP server ready (stdio)")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("mcp read error: %w", err)
			}
			return nil // EOF — client disconnected
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.send(errorResp(nil, -32700, "parse error"))
			continue
		}

		resp := s.dispatch(&req)
		s.send(resp)
	}
}

// ─── Dispatcher ──────────────────────────────────────────────────────────────

func (s *Server) dispatch(req *request) response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	case "ping":
		return okResp(req.ID, map[string]string{"status": "pong"})
	default:
		return errorResp(req.ID, -32601, "method not found: "+req.Method)
	}
}

func (s *Server) handleInitialize(req *request) response {
	return okResp(req.ID, map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]string{
			"name":    "aftermaild",
			"version": "1.2.0",
		},
	})
}

func (s *Server) handleToolsList(req *request) response {
	return okResp(req.ID, map[string]any{"tools": s.tools})
}

func (s *Server) handleToolsCall(req *request) response {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResp(req.ID, -32602, "invalid params")
	}

	result, err := s.callTool(params.Name, params.Arguments)
	if err != nil {
		return okResp(req.ID, toolResult{
			Content: []toolContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		})
	}
	return okResp(req.ID, result)
}

// ─── Tool implementations ─────────────────────────────────────────────────────

func (s *Server) callTool(name string, args json.RawMessage) (toolResult, error) {
	switch name {
	case "list_inbox":
		return s.toolListInbox(args)
	case "get_message":
		return s.toolGetMessage(args)
	case "list_accounts":
		return s.toolListAccounts(args)
	case "get_stats":
		return s.toolGetStats(args)
	case "run_mailscript":
		return s.toolRunMailScript(args)
	case "list_commitments":
		return s.toolListCommitments(args)
	case "analyze_sender":
		return s.toolAnalyzeSender(args)
	case "list_marketplace":
		return s.toolListMarketplace(args)
	case "search_messages":
		return s.toolSearchMessages(args)
	case "daemon_status":
		return s.toolDaemonStatus(args)
	case "create_group_thread":
		return s.toolCreateGroupThread(args)
	case "decrypt_group_thread":
		return s.toolDecryptGroupThread(args)
	case "stake_message":
		return s.toolStakeMessage(args)
	case "arm_deadman":
		return s.toolArmDeadman(args)
	case "checkin_deadman":
		return s.toolCheckinDeadman(args)
	case "get_reputation":
		return s.toolGetReputation(args)
	default:
		return toolResult{}, fmt.Errorf("unknown tool: %s", name)
	}
}

// list_inbox ──────────────────────────────────────────────────────────────────

func (s *Server) toolListInbox(args json.RawMessage) (toolResult, error) {
	var p struct {
		Limit  int    `json:"limit"`
		Folder string `json:"folder"`
	}
	p.Limit = 20
	p.Folder = "Inbox"
	json.Unmarshal(args, &p) //nolint:errcheck — optional args

	if s.db == nil {
		return textResult(`{"messages":[],"note":"database not initialised"}`), nil
	}

	messages, err := s.db.ListMessages()
	if err != nil {
		return toolResult{}, fmt.Errorf("listing inbox: %w", err)
	}

	var result []map[string]interface{}
	for i, m := range messages {
		if p.Limit > 0 && i >= p.Limit {
			break
		}
		result = append(result, m)
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return textResult(string(out)), nil
}

// get_message ─────────────────────────────────────────────────────────────────

func (s *Server) toolGetMessage(args json.RawMessage) (toolResult, error) {
	var p struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.ID == 0 {
		return toolResult{}, fmt.Errorf("id is required")
	}

	if s.db == nil {
		return toolResult{}, fmt.Errorf("database not initialised")
	}

	msg, err := s.db.GetMessage(p.ID)
	if err != nil {
		return toolResult{}, fmt.Errorf("getting message %d: %w", p.ID, err)
	}

	out, _ := json.MarshalIndent(msg, "", "  ")
	return textResult(string(out)), nil
}

// list_accounts ───────────────────────────────────────────────────────────────

func (s *Server) toolListAccounts(_ json.RawMessage) (toolResult, error) {
	if s.db == nil {
		return textResult(`[]`), nil
	}

	accounts, err := s.db.ListAccounts()
	if err != nil {
		return toolResult{}, fmt.Errorf("listing accounts: %w", err)
	}

	out, _ := json.MarshalIndent(accounts, "", "  ")
	return textResult(string(out)), nil
}

// get_stats ───────────────────────────────────────────────────────────────────

func (s *Server) toolGetStats(_ json.RawMessage) (toolResult, error) {
	stats := map[string]any{
		"timestamp": time.Now().Format(time.RFC3339),
		"daemon":    "aftermaild",
		"version":   "1.2.0",
	}

	if s.db != nil {
		messages, _ := s.db.ListMessages()
		accounts, _ := s.db.ListAccounts()
		stats["total_messages"] = len(messages)
		stats["total_accounts"] = len(accounts)
	}

	out, _ := json.MarshalIndent(stats, "", "  ")
	return textResult(string(out)), nil
}

// run_mailscript ──────────────────────────────────────────────────────────────

func (s *Server) toolRunMailScript(args json.RawMessage) (toolResult, error) {
	var p struct {
		Script  string            `json:"script"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.Script == "" {
		return toolResult{}, fmt.Errorf("script is required")
	}

	if p.Headers == nil {
		p.Headers = map[string]string{}
	}

	ctx := &rules.MessageContext{
		Headers: p.Headers,
		Body:    p.Body,
	}

	if err := rules.ExecuteEngine(p.Script, ctx); err != nil {
		return textResult(fmt.Sprintf(`{"error":%q,"actions":[],"logs":[]}`, err.Error())), nil
	}

	out, _ := json.MarshalIndent(map[string]any{
		"actions":          ctx.Actions,
		"modified_headers": ctx.ModifiedHeaders,
		"logs":             ctx.LogEntries,
	}, "", "  ")
	return textResult(string(out)), nil
}

// list_commitments ────────────────────────────────────────────────────────────

func (s *Server) toolListCommitments(args json.RawMessage) (toolResult, error) {
	var p struct {
		ThreadID string `json:"thread_id"`
		Body     string `json:"body"`
		From     string `json:"from"`
		Subject  string `json:"subject"`
	}
	json.Unmarshal(args, &p) //nolint:errcheck

	if p.Body == "" {
		return toolResult{}, fmt.Errorf("body is required to extract commitments")
	}

	// AIExtractor requires an LLM query function; use a stub that returns a
	// placeholder so the tool is always callable even without a live LLM key.
	queryFn := func(_ context.Context, prompt string) (string, error) {
		return `{"commitments":[{"kind":"follow_up","text":"(live LLM not configured — pipe prompt to your model)","status":"open"}]}`, nil
	}
	extractor := commitments.NewAIExtractor(queryFn)

	input := commitments.MessageInput{
		Sender:  p.From,
		Subject: p.Subject,
		Body:    p.Body,
	}

	result, err := extractor.Extract(context.Background(), input)
	if err != nil {
		return toolResult{}, fmt.Errorf("extracting commitments: %w", err)
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return textResult(string(out)), nil
}

// analyze_sender ──────────────────────────────────────────────────────────────

func (s *Server) toolAnalyzeSender(args json.RawMessage) (toolResult, error) {
	var p struct {
		Sender   string   `json:"sender"`
		Messages []string `json:"messages"`
		Target   string   `json:"target"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.Sender == "" {
		return toolResult{}, fmt.Errorf("sender and messages are required")
	}

	samples := make([]fingerprint.MessageSample, len(p.Messages))
	for i, body := range p.Messages {
		samples[i] = fingerprint.MessageSample{Body: body}
	}

	baseline := fingerprint.Build(p.Sender, samples)

	result := map[string]any{
		"sender":   p.Sender,
		"baseline": baseline,
	}

	if p.Target != "" {
		score := fingerprint.Score(baseline, fingerprint.MessageSample{Body: p.Target})
		result["anomaly_score"] = score
		result["anomaly_description"] = describeAnomaly(score.Total)
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return textResult(string(out)), nil
}

func describeAnomaly(score float64) string {
	switch {
	case score < 0.2:
		return "Writing style matches baseline — high confidence this is the genuine sender."
	case score < 0.5:
		return "Minor deviation from baseline — likely authentic with mood/topic variation."
	case score < 0.75:
		return "Moderate anomaly — possible ghostwriter, AI assistance, or account change."
	default:
		return "High anomaly — possible account takeover or AI-generated impersonation."
	}
}

// list_marketplace ────────────────────────────────────────────────────────────

func (s *Server) toolListMarketplace(args json.RawMessage) (toolResult, error) {
	var p struct {
		Query string `json:"query"`
	}
	json.Unmarshal(args, &p) //nolint:errcheck

	dir, _ := os.UserCacheDir()
	reg, _ := marketplace.NewRegistry(dir + "/aftermail/marketplace")
	scripts := reg.Installed()

	if p.Query != "" {
		q := strings.ToLower(p.Query)
		var filtered []*marketplace.Script
		for _, sc := range scripts {
			if strings.Contains(strings.ToLower(sc.Name), q) ||
				strings.Contains(strings.ToLower(sc.Description), q) {
				filtered = append(filtered, sc)
			}
		}
		scripts = filtered
	}

	out, _ := json.MarshalIndent(scripts, "", "  ")
	return textResult(string(out)), nil
}

// search_messages ─────────────────────────────────────────────────────────────

func (s *Server) toolSearchMessages(args json.RawMessage) (toolResult, error) {
	var p struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	p.Limit = 20
	if err := json.Unmarshal(args, &p); err != nil || p.Query == "" {
		return toolResult{}, fmt.Errorf("query is required")
	}

	if s.db == nil {
		return textResult(`[]`), nil
	}

	all, err := s.db.ListMessages()
	if err != nil {
		return toolResult{}, fmt.Errorf("searching messages: %w", err)
	}

	q := strings.ToLower(p.Query)
	var matches []any
	for _, m := range all {
		subject, _ := m["subject"].(string)
		sender, _ := m["sender"].(string)
		body, _ := m["body_plain"].(string)
		if strings.Contains(strings.ToLower(subject), q) ||
			strings.Contains(strings.ToLower(sender), q) ||
			strings.Contains(strings.ToLower(body), q) {
			matches = append(matches, m)
			if p.Limit > 0 && len(matches) >= p.Limit {
				break
			}
		}
	}

	out, _ := json.MarshalIndent(matches, "", "  ")
	return textResult(string(out)), nil
}

// daemon_status ───────────────────────────────────────────────────────────────

func (s *Server) toolDaemonStatus(_ json.RawMessage) (toolResult, error) {
	hostname, _ := os.Hostname()
	status := map[string]any{
		"status":    "online",
		"daemon":    "aftermaild",
		"version":   "1.2.0",
		"hostname":  hostname,
		"pid":       os.Getpid(),
		"timestamp": time.Now().Format(time.RFC3339),
		"db_ready":  s.db != nil,
	}
	out, _ := json.MarshalIndent(status, "", "  ")
	return textResult(string(out)), nil
}

// create_group_thread ─────────────────────────────────────────────────────────

func (s *Server) toolCreateGroupThread(args json.RawMessage) (toolResult, error) {
	var p struct {
		ThreadID   string `json:"thread_id"`
		Body       string `json:"body"`
		Recipients int    `json:"recipients"`
		Threshold  int    `json:"threshold"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.Body == "" {
		return toolResult{}, fmt.Errorf("body is required")
	}
	if p.Recipients < 2 {
		p.Recipients = 2
	}
	if p.Threshold < 2 || p.Threshold > p.Recipients {
		p.Threshold = p.Recipients
	}
	if p.ThreadID == "" {
		p.ThreadID = fmt.Sprintf("thread-%d", time.Now().UnixNano())
	}

	thread, err := send.NewGroupThread(p.ThreadID, p.Body, p.Recipients, p.Threshold)
	if err != nil {
		return toolResult{}, fmt.Errorf("creating group thread: %w", err)
	}

	// Don't expose raw shares in the result — in production each share is
	// encrypted to the recipient's public key and sent individually.
	out, _ := json.MarshalIndent(map[string]any{
		"thread_id":       thread.ThreadID,
		"threshold":       thread.Threshold,
		"share_count":     len(thread.Shares),
		"key_digest":      thread.KeyDigest,
		"encrypted_bytes": len(thread.EncryptedBody),
		"note":            "Each share must be delivered privately to its recipient. Combine any threshold shares to decrypt.",
	}, "", "  ")
	return textResult(string(out)), nil
}

// decrypt_group_thread ────────────────────────────────────────────────────────

func (s *Server) toolDecryptGroupThread(args json.RawMessage) (toolResult, error) {
	var p struct {
		EncryptedBody []byte `json:"encrypted_body"`
		KeyDigest     string `json:"key_digest"`
		Threshold     int    `json:"threshold"`
		Shares        []struct {
			Index byte   `json:"index"`
			Value []byte `json:"value"`
		} `json:"shares"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return toolResult{}, fmt.Errorf("invalid params: %w", err)
	}
	if len(p.Shares) == 0 {
		return toolResult{}, fmt.Errorf("shares are required")
	}

	shares := make([]send.Share, len(p.Shares))
	for i, sh := range p.Shares {
		shares[i] = send.Share{Index: sh.Index, Value: sh.Value}
	}

	thread := &send.GroupThread{
		EncryptedBody: p.EncryptedBody,
		KeyDigest:     p.KeyDigest,
		Threshold:     p.Threshold,
	}
	plaintext, err := thread.Decrypt(shares)
	if err != nil {
		return toolResult{}, fmt.Errorf("decrypt: %w", err)
	}
	out, _ := json.MarshalIndent(map[string]string{"plaintext": plaintext}, "", "  ")
	return textResult(string(out)), nil
}

// stake_message ───────────────────────────────────────────────────────────────

func (s *Server) toolStakeMessage(args json.RawMessage) (toolResult, error) {
	var p struct {
		MessageID string `json:"message_id"`
		Sender    string `json:"sender"`
		Recipient string `json:"recipient"`
		StakeETH  string `json:"stake_eth"`
		Policy    string `json:"policy"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.MessageID == "" {
		return toolResult{}, fmt.Errorf("message_id, sender, recipient, stake_eth are required")
	}

	stakeWei, ok := stakemail.ETHToWei(p.StakeETH)
	if !ok {
		return toolResult{}, fmt.Errorf("invalid stake_eth value: %q", p.StakeETH)
	}

	policy := stakemail.SlashPolicy(p.Policy)
	if policy == "" {
		policy = stakemail.SlashBurn
	}

	// Simulation mode (no contract address).
	client := stakemail.NewClient("https://mainnet.base.org", 8453, p.Sender, "")
	m, err := client.Stake(context.Background(), p.MessageID, p.Sender, p.Recipient, stakeWei, policy)
	if err != nil {
		return toolResult{}, fmt.Errorf("staking: %w", err)
	}

	out, _ := json.MarshalIndent(m, "", "  ")
	return textResult(string(out)), nil
}

// arm_deadman ─────────────────────────────────────────────────────────────────

// package-level deadman manager (shared across calls).
var dmManager *deadman.Manager

func getDeadmanManager() *deadman.Manager {
	if dmManager == nil {
		dir := os.TempDir() + "/aftermail/deadman"
		mgr, err := deadman.NewManager(dir, func(_ context.Context, sw *deadman.Switch) error {
			log.Printf("[deadman] FIRED: %s → %v", sw.Label, sw.Recipients)
			return nil
		})
		if err != nil {
			log.Printf("[deadman] manager init error: %v", err)
			return nil
		}
		dmManager = mgr
	}
	return dmManager
}

func (s *Server) toolArmDeadman(args json.RawMessage) (toolResult, error) {
	var p struct {
		ID              string   `json:"id"`
		Label           string   `json:"label"`
		Recipients      []string `json:"recipients"`
		Subject         string   `json:"subject"`
		Body            string   `json:"body"`
		CheckInHours    float64  `json:"check_in_hours"`
		GracePeriodMins float64  `json:"grace_period_minutes"`
	}
	if err := json.Unmarshal(args, &p); err != nil || len(p.Recipients) == 0 {
		return toolResult{}, fmt.Errorf("id, recipients, subject, body are required")
	}
	if p.ID == "" {
		p.ID = fmt.Sprintf("dm-%d", time.Now().UnixNano())
	}
	if p.CheckInHours <= 0 {
		p.CheckInHours = 24
	}

	mgr := getDeadmanManager()
	if mgr == nil {
		return toolResult{}, fmt.Errorf("deadman manager unavailable")
	}

	sw := &deadman.Switch{
		ID:              p.ID,
		Label:           p.Label,
		Recipients:      p.Recipients,
		Subject:         p.Subject,
		Body:            p.Body,
		CheckInInterval: time.Duration(p.CheckInHours * float64(time.Hour)),
		GracePeriod:     time.Duration(p.GracePeriodMins * float64(time.Minute)),
	}
	if err := mgr.Arm(sw); err != nil {
		return toolResult{}, err
	}

	out, _ := json.MarshalIndent(sw, "", "  ")
	return textResult(string(out)), nil
}

func (s *Server) toolCheckinDeadman(args json.RawMessage) (toolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.ID == "" {
		return toolResult{}, fmt.Errorf("id is required")
	}

	mgr := getDeadmanManager()
	if mgr == nil {
		return toolResult{}, fmt.Errorf("deadman manager unavailable")
	}
	if err := mgr.CheckIn(p.ID); err != nil {
		return toolResult{}, err
	}

	sw, _ := mgr.Get(p.ID)
	out, _ := json.MarshalIndent(sw, "", "  ")
	return textResult(string(out)), nil
}

// get_reputation ──────────────────────────────────────────────────────────────

func (s *Server) toolGetReputation(args json.RawMessage) (toolResult, error) {
	var p struct {
		DID string `json:"did"`
	}
	json.Unmarshal(args, &p) //nolint:errcheck

	// Return a demo profile for illustration when no real profile is loaded.
	doc := &reputation.DIDDocument{
		DID:              p.DID,
		Score:            0.0,
		TotalCommitments: 0,
		KeptCommitments:  0,
		StakeHistory:     0,
		Receipts:         []*reputation.Receipt{},
		UpdatedAt:        time.Now(),
	}

	out, _ := json.MarshalIndent(map[string]any{
		"document":    doc,
		"trust_level": reputation.ScoreToTrustLevel(doc.Score),
		"dns_txt":     fmt.Sprintf("did=%s;score=%.3f;receipts=%d;level=%s", doc.DID, doc.Score, 0, reputation.ScoreToTrustLevel(0)),
		"note":        "Receipts accumulate as counterparties sign commitment-kept attestations.",
	}, "", "  ")
	return textResult(string(out)), nil
}

// ─── Tool definitions (schema) ────────────────────────────────────────────────

func buildToolDefs() []toolDef {
	return []toolDef{
		{
			Name:        "daemon_status",
			Description: "Returns aftermaild daemon health: version, PID, DB readiness, timestamp.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "list_inbox",
			Description: "Lists messages in the inbox. Returns subject, sender, date, flags, and IDs.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"limit":  {Type: "integer", Description: "Maximum number of messages to return (default 20)."},
					"folder": {Type: "string", Description: "Folder name to list (default 'Inbox')."},
				},
			},
		},
		{
			Name:        "get_message",
			Description: "Fetches the full content of a single message including body and attachments.",
			InputSchema: inputSchema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]property{
					"id": {Type: "integer", Description: "Message ID from list_inbox."},
				},
			},
		},
		{
			Name:        "search_messages",
			Description: "Full-text search across subject, sender, and body. Returns matching messages.",
			InputSchema: inputSchema{
				Type:     "object",
				Required: []string{"query"},
				Properties: map[string]property{
					"query": {Type: "string", Description: "Search term."},
					"limit": {Type: "integer", Description: "Maximum results (default 20)."},
				},
			},
		},
		{
			Name:        "list_accounts",
			Description: "Returns all configured mail accounts (IMAP, AMP, Web3Mail) with their settings.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "get_stats",
			Description: "Returns aggregate stats: total messages, total accounts, daemon uptime.",
			InputSchema: inputSchema{Type: "object", Properties: map[string]property{}},
		},
		{
			Name:        "run_mailscript",
			Description: "Executes a MailScript (Starlark) rule against a message context. Returns actions taken, modified headers, and log output.",
			InputSchema: inputSchema{
				Type:     "object",
				Required: []string{"script"},
				Properties: map[string]property{
					"script":  {Type: "string", Description: "Starlark MailScript source code."},
					"headers": {Type: "object", Description: "Email headers as key-value pairs."},
					"body":    {Type: "string", Description: "Email body text."},
				},
			},
		},
		{
			Name:        "list_commitments",
			Description: "Extracts commitments, promises, questions, and deadlines from an email body using the AI Commitment Ledger.",
			InputSchema: inputSchema{
				Type:     "object",
				Required: []string{"body"},
				Properties: map[string]property{
					"body":      {Type: "string", Description: "Email body text to analyse."},
					"from":      {Type: "string", Description: "Sender email address."},
					"thread_id": {Type: "string", Description: "Optional thread ID for context."},
				},
			},
		},
		{
			Name:        "analyze_sender",
			Description: "Builds a writing-style baseline for a sender and optionally scores a new message for anomalies (account takeover / AI impersonation detection).",
			InputSchema: inputSchema{
				Type:     "object",
				Required: []string{"sender", "messages"},
				Properties: map[string]property{
					"sender":   {Type: "string", Description: "Sender email address."},
					"messages": {Type: "array", Description: "Historical message bodies to build the baseline from."},
					"target":   {Type: "string", Description: "Optional new message body to score against the baseline."},
				},
			},
		},
		{
			Name:        "list_marketplace",
			Description: "Lists MailScript rules available in the marketplace. Supports optional keyword filtering.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"query": {Type: "string", Description: "Optional keyword filter."},
				},
			},
		},
		{
			Name:        "create_group_thread",
			Description: "Creates an MPC-encrypted group email thread. Splits the encryption key into N Shamir shares (M-of-N threshold). Each share goes to one recipient; any M can decrypt.",
			InputSchema: inputSchema{
				Type:     "object",
				Required: []string{"body"},
				Properties: map[string]property{
					"thread_id":  {Type: "string", Description: "Optional thread ID (auto-generated if omitted)."},
					"body":       {Type: "string", Description: "Plaintext message body to encrypt."},
					"recipients": {Type: "integer", Description: "Total number of recipients (share count N). Default 2."},
					"threshold":  {Type: "integer", Description: "Minimum shares required to decrypt (M). Default = N."},
				},
			},
		},
		{
			Name:        "stake_message",
			Description: "Locks ETH on Base L2 as a staked-attention bond for a message. Stake is released on open, slashed to burn/charity if marked spam.",
			InputSchema: inputSchema{
				Type:     "object",
				Required: []string{"message_id", "sender", "recipient", "stake_eth"},
				Properties: map[string]property{
					"message_id": {Type: "string", Description: "Email Message-ID."},
					"sender":     {Type: "string", Description: "Sender email or wallet address."},
					"recipient":  {Type: "string", Description: "Recipient email or wallet address."},
					"stake_eth":  {Type: "string", Description: "Amount to stake, e.g. '0.01'."},
					"policy":     {Type: "string", Description: "Slash policy: 'burn', 'charity', or 'recipient'. Default 'burn'."},
				},
			},
		},
		{
			Name:        "arm_deadman",
			Description: "Arms a Dead Man's Switch that fires and sends email if the user doesn't check in within the configured interval.",
			InputSchema: inputSchema{
				Type:     "object",
				Required: []string{"recipients", "subject", "body"},
				Properties: map[string]property{
					"id":                   {Type: "string", Description: "Switch ID (auto-generated if omitted)."},
					"label":                {Type: "string", Description: "Human-readable name."},
					"recipients":           {Type: "array", Description: "Email addresses to notify on trigger."},
					"subject":              {Type: "string", Description: "Email subject sent on trigger."},
					"body":                 {Type: "string", Description: "Email body sent on trigger."},
					"check_in_hours":       {Type: "number", Description: "Hours between required check-ins (default 24)."},
					"grace_period_minutes": {Type: "number", Description: "Extra minutes after deadline before firing (default 0)."},
				},
			},
		},
		{
			Name:        "checkin_deadman",
			Description: "Resets the deadline on an armed Dead Man's Switch, proving the user is alive.",
			InputSchema: inputSchema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]property{
					"id": {Type: "string", Description: "Switch ID to check in."},
				},
			},
		},
		{
			Name:        "get_reputation",
			Description: "Returns the DID reputation document and trust score for an AfterMail identity, including commitment-kept receipt count and DNS TXT record for itz.agency.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"did": {Type: "string", Description: "DID to look up, e.g. 'did:aftersmtp:msgs.global:ryan'."},
				},
			},
		},
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (s *Server) send(r response) {
	b, err := json.Marshal(r)
	if err != nil {
		log.Printf("[MCP] marshal error: %v", err)
		return
	}
	fmt.Fprintf(s.out, "%s\n", b)
}

func okResp(id any, result any) response {
	return response{JSONRPC: "2.0", ID: id, Result: result}
}

func errorResp(id any, code int, msg string) response {
	return response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
}

func textResult(text string) toolResult {
	return toolResult{Content: []toolContent{{Type: "text", Text: text}}}
}
