package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/afterdarksys/aftermail/pkg/mcp"
)

func send(t *testing.T, method string, params string) map[string]any {
	t.Helper()
	line := `{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + "}\n"
	var out bytes.Buffer
	srv := mcp.New(nil, strings.NewReader(line), &out)
	srv.Run(context.Background())
	var result map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &result); err != nil {
		t.Fatalf("unmarshal: %v — raw: %s", err, out.String())
	}
	return result
}

func TestInitialize(t *testing.T) {
	resp := send(t, "initialize", "{}")
	if resp["error"] != nil {
		t.Fatalf("got error: %v", resp["error"])
	}
	res := resp["result"].(map[string]any)
	if res["protocolVersion"] != "2024-11-05" {
		t.Errorf("unexpected protocolVersion: %v", res["protocolVersion"])
	}
}

func TestToolsList(t *testing.T) {
	resp := send(t, "tools/list", "{}")
	if resp["error"] != nil {
		t.Fatalf("got error: %v", resp["error"])
	}
	res := resp["result"].(map[string]any)
	tools := res["tools"].([]any)
	if len(tools) < 8 {
		t.Errorf("expected at least 8 tools, got %d", len(tools))
	}
}

func TestDaemonStatus(t *testing.T) {
	resp := send(t, "tools/call", `{"name":"daemon_status","arguments":{}}`)
	if resp["error"] != nil {
		t.Fatalf("got error: %v", resp["error"])
	}
	res := resp["result"].(map[string]any)
	content := res["content"].([]any)
	if len(content) == 0 {
		t.Fatal("expected content in result")
	}
	text := content[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "aftermaild") {
		t.Errorf("unexpected status text: %s", text)
	}
}

func TestListInboxNilDB(t *testing.T) {
	resp := send(t, "tools/call", `{"name":"list_inbox","arguments":{"limit":5}}`)
	if resp["error"] != nil {
		t.Fatalf("got error: %v", resp["error"])
	}
	// Should return gracefully even with nil DB
	res := resp["result"].(map[string]any)
	if res["isError"] == true {
		t.Errorf("expected graceful nil-DB handling, got isError")
	}
}

func TestRunMailScript(t *testing.T) {
	script := `def evaluate():\n    accept()\n`
	arg := `{"name":"run_mailscript","arguments":{"script":"` + script + `","headers":{"From":"alice@example.com"},"body":"Hello"}}`
	resp := send(t, "tools/call", arg)
	if resp["error"] != nil {
		t.Fatalf("got error: %v", resp["error"])
	}
}

func TestUnknownMethod(t *testing.T) {
	resp := send(t, "nope/nope", "{}")
	if resp["error"] == nil {
		t.Fatal("expected error for unknown method")
	}
}
