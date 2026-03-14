package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestMCPStartsFromDifferentCwd verifies that browser-cmd --mcp starts without
// path errors when run from a directory other than browser/ (e.g. when
// launched by Claude Code from the user's project). Paths must resolve
// relative to the executable, not cwd.
func TestMCPStartsFromDifferentCwd(t *testing.T) {
	// Build binary in browser dir (always rebuild to pick up latest code)
	browserDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	buildCmd := exec.Command("go", "build", "-trimpath", "-o", "browser-cmd", ".")
	buildCmd.Dir = browserDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	binaryPath := filepath.Join(browserDir, "browser-cmd")

	// Run from /tmp (simulates Claude Code running from user's project)
	cmd := exec.Command(binaryPath, "--mcp")
	cmd.Dir = "/tmp"
	cmd.Env = os.Environ()

	// Start and wait briefly — MCP blocks on stdio, we just need to verify
	// it doesn't exit immediately with "JS_SCRIPTS_PATH does not exist"
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("browser-cmd exited immediately (likely path error): %v", err)
		}
	case <-time.After(2 * time.Second):
		// Still running after 2s — path resolution worked
		_ = cmd.Process.Kill()
		<-done
	}
}

func TestObservationsDir_Unset(t *testing.T) {
	t.Setenv("OBSERVATIONS_DIR", "")
	if got := observationsDir(); got != "" {
		t.Errorf("expected empty string when OBSERVATIONS_DIR is unset, got %q", got)
	}
}

func TestObservationsDir_Set(t *testing.T) {
	t.Setenv("OBSERVATIONS_DIR", "/tmp/obs")
	if got := observationsDir(); got != "/tmp/obs" {
		t.Errorf("expected /tmp/obs, got %q", got)
	}
}
