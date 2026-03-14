package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNewConfigDefaults(t *testing.T) {
	cfg := NewConfig()
	if cfg.WSPort != 9001 {
		t.Errorf("expected WSPort 9001, got %d", cfg.WSPort)
	}
	if cfg.JSScriptsPath != "./resources/js_scripts" {
		t.Errorf("expected default JSScriptsPath './resources/js_scripts', got %q", cfg.JSScriptsPath)
	}
}

func TestLoadWSPortFromEnv(t *testing.T) {
	t.Setenv("WS_PORT", "9099")
	t.Setenv("JS_SCRIPTS_PATH", t.TempDir())

	cfg, err := NewConfig().Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.WSPort != 9099 {
		t.Errorf("expected WSPort 9099, got %d", cfg.WSPort)
	}
}

func TestLoadInvalidWSPort(t *testing.T) {
	t.Setenv("WS_PORT", "not-a-number")

	_, err := NewConfig().Load()
	if err == nil {
		t.Fatal("expected error for invalid WS_PORT")
	}
}

func TestLoadPortBoundaryValues(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		port    string
		wantErr bool
	}{
		{"1023", true},
		{"1024", false},
		{"65535", false},
		{"65536", true},
		{"100", true},
	}
	for _, tc := range cases {
		t.Run("port="+tc.port, func(t *testing.T) {
			t.Setenv("WS_PORT", tc.port)
			t.Setenv("JS_SCRIPTS_PATH", dir)
			_, err := NewConfig().Load()
			if tc.wantErr && err == nil {
				t.Errorf("expected error for port %s", tc.port)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for port %s: %v", tc.port, err)
			}
		})
	}
}

func TestLoadMissingScriptsPath(t *testing.T) {
	t.Setenv("JS_SCRIPTS_PATH", "/nonexistent/path/xyz")

	_, err := NewConfig().Load()
	if err == nil {
		t.Fatal("expected error for missing scripts path")
	}
}

func TestLoadJSScriptsPathIsAbsolute(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("JS_SCRIPTS_PATH", dir)

	cfg, err := NewConfig().Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(cfg.JSScriptsPath) {
		t.Errorf("JSScriptsPath should be absolute, got %q", cfg.JSScriptsPath)
	}
}

func TestLoadBothErrors(t *testing.T) {
	t.Setenv("WS_PORT", "100")
	t.Setenv("JS_SCRIPTS_PATH", "/nonexistent/xyz")

	_, err := NewConfig().Load()
	if err == nil {
		t.Fatal("expected error when both port and path are invalid")
	}
	msg := err.Error()
	if !strings.Contains(msg, "WS_PORT") {
		t.Errorf("expected WS_PORT mention in error: %s", msg)
	}
}
