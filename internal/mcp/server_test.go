package mcpserver_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paoloandrisani/browser-mcp-extension/internal/api"
	mcpserver "github.com/paoloandrisani/browser-mcp-extension/internal/mcp"
	"github.com/paoloandrisani/browser-mcp-extension/internal/observation"
)

func buildTestHandler(t *testing.T) *api.Handler {
	t.Helper()
	dir := t.TempDir()
	store, _ := observation.NewStore(dir)
	scriptsDir := filepath.Join(dir, "scripts")
	os.MkdirAll(scriptsDir, 0o755)
	execute := func(code string) (bool, interface{}, string, error) { return true, "ok", "", nil }
	executeFile := func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		return true, "ok", "", nil
	}
	screenshot := func() (string, error) { return "data:image/png;base64,abc", nil }
	tabs := func(action string, index *int, url string) (bool, interface{}, string, error) {
		return true, nil, "", nil
	}
	return api.NewHandler(store, execute, executeFile, screenshot, tabs, scriptsDir, "")
}

func TestNewServerRegisters18Tools(t *testing.T) {
	h := buildTestHandler(t)
	s := mcpserver.NewServer(h)
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	tools := s.ListTools()
	if len(tools) != 18 {
		t.Errorf("expected 18 tools, got %d", len(tools))
		for name := range tools {
			t.Logf("  - %s", name)
		}
	}
}

func TestToolResultMarshal(t *testing.T) {
	// Indirect test via NewServer creation — ensures toolResult helper does not panic.
	h := buildTestHandler(t)
	s := mcpserver.NewServer(h)
	if s == nil {
		t.Fatal("expected non-nil server")
	}
}
