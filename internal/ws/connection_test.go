package ws

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestMessageMarshalExecute(t *testing.T) {
	msg := Message{Type: "execute", Code: "return 1+1"}
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Message
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Type != "execute" {
		t.Errorf("expected type 'execute', got %q", out.Type)
	}
	if out.Code != "return 1+1" {
		t.Errorf("expected code 'return 1+1', got %q", out.Code)
	}
}

func TestMessageMarshalScreenshot(t *testing.T) {
	msg := Message{Type: "screenshot"}
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["type"] != "screenshot" {
		t.Errorf("expected type 'screenshot', got %v", m["type"])
	}
	if _, ok := m["code"]; ok {
		t.Error("'code' should be omitted for screenshot message")
	}
	if _, ok := m["action"]; ok {
		t.Error("'action' should be omitted for screenshot message")
	}
}

func TestMessageMarshalTabsWithIndex(t *testing.T) {
	idx := 2
	msg := Message{Type: "tabs", Action: "close", Index: &idx}
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Message
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Action != "close" {
		t.Errorf("expected action 'close', got %q", out.Action)
	}
	if out.Index == nil || *out.Index != 2 {
		t.Errorf("expected index 2, got %v", out.Index)
	}
}

func TestMessageMarshalTabsNilIndexOmitted(t *testing.T) {
	msg := Message{Type: "tabs", Action: "list"}
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["index"]; ok {
		t.Error("'index' should be omitted when nil (omitempty)")
	}
}

func TestExecuteNoConnection(t *testing.T) {
	c := NewConnection()
	_, _, _, err := c.Execute("return 1")
	if err == nil {
		t.Fatal("expected error when no connection")
	}
}

func TestScreenshotNoConnection(t *testing.T) {
	c := NewConnection()
	_, err := c.Screenshot()
	if err == nil {
		t.Fatal("expected error when no connection")
	}
}

func TestTabsNoConnection(t *testing.T) {
	c := NewConnection()
	_, _, _, err := c.Tabs("list", nil, "")
	if err == nil {
		t.Fatal("expected error when no connection")
	}
}

func TestExecuteFile_MessageFields(t *testing.T) {
	params := map[string]interface{}{"action": "click", "elementIndex": float64(0)}
	msg := Message{
		Type:       "execute",
		Code:       "console.log('fallback');",
		ScriptFile: "scripts/interact.js",
		Params:     params,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded["type"] != "execute" {
		t.Errorf("expected type=execute, got %v", decoded["type"])
	}
	if decoded["code"] != "console.log('fallback');" {
		t.Errorf("expected code field, got %v", decoded["code"])
	}
	if decoded["scriptFile"] != "scripts/interact.js" {
		t.Errorf("expected scriptFile=scripts/interact.js, got %v", decoded["scriptFile"])
	}
	p, ok := decoded["params"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected params to be a map, got %T", decoded["params"])
	}
	if p["action"] != "click" {
		t.Errorf("expected params.action=click, got %v", p["action"])
	}
}

func TestExecuteFile_OmitsEmptyFields(t *testing.T) {
	msg := Message{Type: "execute", Code: "return 1;"}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, exists := decoded["scriptFile"]; exists {
		t.Error("scriptFile should be omitted when empty")
	}
	if _, exists := decoded["params"]; exists {
		t.Error("params should be omitted when nil")
	}
}

func TestExecuteFileNoConnection(t *testing.T) {
	c := NewConnection()
	_, _, _, err := c.ExecuteFile("scripts/test.js", "return 1", nil)
	if err == nil {
		t.Fatal("expected error when no connection")
	}
}

// freePort returns an available TCP port for testing.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func TestStartServer_TokenRejectsUnauthorized(t *testing.T) {
	c := NewConnection()
	port := freePort(t)
	c.StartServer(port, "secret123")
	time.Sleep(50 * time.Millisecond) // let server bind

	// Attempt connection without token — should get 401
	url := fmt.Sprintf("ws://127.0.0.1:%d/", port)
	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		t.Fatal("expected dial to fail without token")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestStartServer_TokenRejectsWrongToken(t *testing.T) {
	c := NewConnection()
	port := freePort(t)
	c.StartServer(port, "correct-token")
	time.Sleep(50 * time.Millisecond)

	url := fmt.Sprintf("ws://127.0.0.1:%d/?token=wrong-token", port)
	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		t.Fatal("expected dial to fail with wrong token")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestStartServer_TokenAcceptsCorrectToken(t *testing.T) {
	c := NewConnection()
	port := freePort(t)
	c.StartServer(port, "correct-token")
	time.Sleep(50 * time.Millisecond)

	url := fmt.Sprintf("ws://127.0.0.1:%d/?token=correct-token", port)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("expected successful dial with correct token, got: %v", err)
	}
	conn.Close()
}

func TestStartServer_EmptyTokenAcceptsAll(t *testing.T) {
	c := NewConnection()
	port := freePort(t)
	c.StartServer(port, "")
	time.Sleep(50 * time.Millisecond)

	url := fmt.Sprintf("ws://127.0.0.1:%d/", port)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("expected successful dial with no token configured, got: %v", err)
	}
	conn.Close()
}

func TestReadyChannelNotNil(t *testing.T) {
	c := NewConnection()
	ch := c.Ready()
	if ch == nil {
		t.Fatal("expected non-nil ready channel")
	}
	// Channel must not be closed before the extension connects.
	select {
	case <-ch:
		t.Fatal("ready channel should not be closed before extension connects")
	default:
	}
}
