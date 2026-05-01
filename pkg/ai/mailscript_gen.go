package ai

import (
	"context"
	"fmt"
	"strings"
)

// mailscriptSystemPrompt teaches the model exactly what MailScript looks like.
const mailscriptSystemPrompt = `You are an expert MailScript engineer. MailScript uses Starlark (Python-like syntax).
Generate ONLY the Starlark script — no explanations, no markdown fences, no comments unless they aid clarity.

Available functions:
  accept()                          — deliver the message
  discard()                         — silently drop it
  drop()                            — forcefully drop it
  bounce()                          — bounce back to sender
  quarantine()                      — move to quarantine
  fileinto(folder)                  — move to named folder
  add_to_next_digest()              — add to daily digest
  auto_reply(text)                  — send automated reply
  divert_to(email)                  — redirect to address
  screen_to(email)                  — send copy for screening
  force_second_pass(mailserver)     — route to another server
  reply_with_smtp_error(code)       — reply with SMTP error
  add_header(name, value)           — add/set a header
  log_entry(message)                — write a log entry

  get_header(name)                  — get header value (string)
  search_body(text)                 — True if text found in body
  regex_match(pattern, text)        — True if regex matches

  getmimetype()                     — MIME type string
  getspamscore()                    — float 0.0–10.0
  getvirusstatus()                  — "clean" / "infected" / "unknown"
  body_size()                       — int bytes
  header_size()                     — int bytes
  get_recipient_did()               — recipient DID string

  skip_malware_check(sender)
  skip_spam_check(sender)
  skip_whitelist_check(ip)

  set_dlp(mode, target)             — mode: "always"/"sometimes"; target: user/domain
  skip_dlp(mode, target)

  get_sender_ip()                   — string
  get_sender_domain()               — string
  dns_check(domain)                 — bool
  rbl_check(ip, rbl_server="")      — bool
  valid_mx(domain)                  — bool
  mx_in_rbl(domain, rbl_server="")  — bool

Rules:
- Always end with a terminal action (accept, discard, bounce, quarantine, fileinto).
- Use guard clauses — check conditions first, fall through to accept() at the end.
- Keep scripts readable with meaningful variable names.
- Use regex_match for complex patterns, search_body for simple substring checks.
`

// GenerateMailScript converts a plain-English description into a MailScript Starlark program.
func (a *Assistant) GenerateMailScript(ctx context.Context, description string) (string, error) {
	prompt := fmt.Sprintf(`Generate a MailScript (Starlark) rule for the following requirement:

%s

Return ONLY the Starlark code. No markdown. No explanation.`, description)

	result, err := a.queryWithSystem(ctx, mailscriptSystemPrompt, prompt)
	if err != nil {
		return "", fmt.Errorf("mailscript generation failed: %w", err)
	}

	// Strip any accidental markdown fences a model might sneak in
	result = strings.TrimSpace(result)
	result = strings.TrimPrefix(result, "```starlark")
	result = strings.TrimPrefix(result, "```python")
	result = strings.TrimPrefix(result, "```")
	result = strings.TrimSuffix(result, "```")
	return strings.TrimSpace(result), nil
}

// RefineMailScript takes an existing script and a change description, returns improved script.
func (a *Assistant) RefineMailScript(ctx context.Context, existingScript, changeDescription string) (string, error) {
	prompt := fmt.Sprintf(`Here is an existing MailScript (Starlark) rule:

%s

Please modify it to also handle the following requirement:

%s

Return ONLY the updated Starlark code. No markdown. No explanation.`, existingScript, changeDescription)

	result, err := a.queryWithSystem(ctx, mailscriptSystemPrompt, prompt)
	if err != nil {
		return "", fmt.Errorf("mailscript refinement failed: %w", err)
	}

	result = strings.TrimSpace(result)
	result = strings.TrimPrefix(result, "```starlark")
	result = strings.TrimPrefix(result, "```python")
	result = strings.TrimPrefix(result, "```")
	result = strings.TrimSuffix(result, "```")
	return strings.TrimSpace(result), nil
}

// ExplainMailScript returns a plain-English explanation of what a script does.
func (a *Assistant) ExplainMailScript(ctx context.Context, script string) (string, error) {
	prompt := fmt.Sprintf(`Explain what the following MailScript (Starlark) rule does in plain English.
Be concise — 1-3 sentences maximum. Focus on what mail gets filtered and what happens to it.

%s`, script)

	return a.queryWithSystem(ctx, mailscriptSystemPrompt, prompt)
}

// queryWithSystem calls the AI with both a system prompt and a user prompt.
func (a *Assistant) queryWithSystem(ctx context.Context, system, user string) (string, error) {
	switch a.Provider {
	case ProviderAnthropic:
		return a.queryAnthropicWithSystem(ctx, system, user)
	case ProviderOpenRouter:
		return a.queryOpenRouterWithSystem(ctx, system, user)
	default:
		return "", fmt.Errorf("unsupported provider: %s", a.Provider)
	}
}
