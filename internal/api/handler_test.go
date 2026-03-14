package api

import (
	"fmt"
	"github.com/paoloandrisani/browser-mcp-extension/internal/observation"
	"os"
	"path/filepath"
	"testing"
)

// ── Test helpers ─────────────────────────────────────────────────────────

func mkdirAll(path string) error { return os.MkdirAll(path, 0o755) }

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// mockExecute returns success=true with the given data on every call.
func mockExecute(data string) ExecuteFn {
	return func(code string) (bool, interface{}, string, error) {
		return true, data, "", nil
	}
}

// mockExecuteFile returns success=true with the given data on every call.
func mockExecuteFile(data string) ExecuteFileFn {
	return func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		return true, data, "", nil
	}
}

func mockScreenshot() ScreenshotFn {
	return func() (string, error) {
		return "data:image/png;base64,abc123", nil
	}
}

func mockTabs() TabsFn {
	return func(action string, index *int, url string) (bool, interface{}, string, error) {
		return true, []map[string]interface{}{{"index": 0, "url": "https://example.com", "active": true}}, "", nil
	}
}

func setupHandler(t *testing.T, executeData string) *Handler {
	t.Helper()
	dir := t.TempDir()
	store, err := observation.NewStore(dir)
	if err != nil {
		t.Fatalf("observation.NewStore: %v", err)
	}
	scriptsDir := filepath.Join(dir, "scripts")
	if err := mkdirAll(scriptsDir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	snapshotJSON := `{"url":"https://example.com","title":"Test","timestamp":"2026-01-01T00:00:00Z","interactiveElements":[],"totalInteractiveElements":0,"visibleText":"hello","sections":[]}`
	writeTestFile(t, filepath.Join(scriptsDir, "observe.js"), `return '`+snapshotJSON+`';`)
	writeTestFile(t, filepath.Join(scriptsDir, "navigate.js"), `return "navigated";`)
	writeTestFile(t, filepath.Join(scriptsDir, "navigate_back.js"), `return "went back";`)
	return NewHandler(store, mockExecute(executeData), mockExecuteFile(executeData), mockScreenshot(), mockTabs(), scriptsDir, "")
}

func TestHandlerGetStateEmpty(t *testing.T) {
	h := setupHandler(t, "")
	snap, err := h.GetState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap != nil {
		t.Error("expected nil snapshot when no observations exist")
	}
}

func TestHandlerListScripts(t *testing.T) {
	h := setupHandler(t, "")
	scripts, err := h.ListScripts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// navigate.js and navigate_back.js should appear; observe.js should be excluded
	found := map[string]bool{}
	for _, s := range scripts {
		found[s.Name] = true
	}
	if !found["navigate.js"] {
		t.Error("expected navigate.js in scripts list")
	}
	if found["observe.js"] {
		t.Error("observe.js should be excluded from scripts list")
	}
}

func TestHandlerWrapCodeNoParams(t *testing.T) {
	wrapped, err := WrapCode("return 1;", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wrapped == "" {
		t.Error("expected non-empty wrapped code")
	}
}

func TestHandlerWrapCodeWithParams(t *testing.T) {
	params := map[string]interface{}{"key": "value"}
	wrapped, err := WrapCode("return params.key;", "", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wrapped == "" {
		t.Error("expected non-empty wrapped code")
	}
}

func TestHandlerBrowserScreenshot(t *testing.T) {
	h := setupHandler(t, "")
	url, err := h.BrowserScreenshot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "data:image/png;base64,abc123" {
		t.Errorf("unexpected dataURL: %s", url)
	}
}

func TestHandlerBrowserTabs(t *testing.T) {
	h := setupHandler(t, "")
	ok, data, _, err := h.BrowserTabs("list", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected success=true")
	}
	if data == nil {
		t.Error("expected non-nil data")
	}
}

func TestHandlerNavigateBack(t *testing.T) {
	h := setupHandler(t, "went back")
	result, snap, err := h.NavigateBack()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
	// snap may be nil if the mock observe returns invalid JSON; that is acceptable here.
	_ = snap
}

func TestHandlerGetStateFromDisk(t *testing.T) {
	dir := t.TempDir()

	// Create and save via store1.
	store1, _ := observation.NewStore(dir)
	snapJSON := `{"url":"https://example.com","title":"Test","timestamp":"2026-01-01T00:00:00Z","interactiveElements":[],"totalInteractiveElements":0,"visibleText":"","sections":[]}`
	_, err := store1.Save(snapJSON, "test", "", true)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	// New handler with same dir — memory is empty, disk has the snapshot.
	scriptsDir := t.TempDir()
	store2, _ := observation.NewStore(dir)
	h := NewHandler(store2, mockExecute(""), mockExecuteFile(""), mockScreenshot(), mockTabs(), scriptsDir, "")

	snap, err := h.GetState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap == nil {
		t.Fatal("expected snapshot from disk")
	}
}

func TestHandlerExecuteScriptNotFound(t *testing.T) {
	h := setupHandler(t, "")
	_, _, _, err := h.ExecuteScript("nonexistent_script.js", nil)
	if err == nil {
		t.Fatal("expected error for missing script")
	}
}

func TestHandlerExecuteScriptPathTraversal(t *testing.T) {
	h := setupHandler(t, "")
	_, _, _, err := h.ExecuteScript("../../../etc/passwd", nil)
	if err == nil {
		t.Fatal("expected error for path traversal attempt")
	}
}

func TestHandlerBrowserScreenshotError(t *testing.T) {
	dir := t.TempDir()
	store, _ := observation.NewStore(dir)
	scriptsDir := t.TempDir()
	errFn := func() (string, error) { return "", fmt.Errorf("screenshot unavailable") }
	h := NewHandler(store, mockExecute(""), mockExecuteFile(""), errFn, mockTabs(), scriptsDir, "")
	_, err := h.BrowserScreenshot()
	if err == nil {
		t.Fatal("expected error from screenshot function")
	}
}

func TestHandlerNavigateToInvalidScheme(t *testing.T) {
	h := setupHandler(t, "")
	err := h.NavigateTo("javascript:alert(1)")
	if err == nil {
		t.Fatal("expected error for javascript: scheme")
	}
	err = h.NavigateTo("file:///etc/passwd")
	if err == nil {
		t.Fatal("expected error for file: scheme")
	}
}

func TestExecuteScript_SendsScriptFile(t *testing.T) {
	dir := t.TempDir()
	store, _ := observation.NewStore(dir)
	scriptsDir := filepath.Join(dir, "scripts")
	mkdirAll(scriptsDir)
	writeTestFile(t, filepath.Join(scriptsDir, "interact.js"), `return 'done';`)

	var capturedScriptFile string
	executeFileFn := func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		capturedScriptFile = scriptFile
		return true, "done", "", nil
	}
	h := NewHandler(store, mockExecute(""), executeFileFn, mockScreenshot(), mockTabs(), scriptsDir, "")

	_, _, _, err := h.ExecuteScript("interact.js", map[string]interface{}{"action": "click"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedScriptFile != "scripts/interact.js" {
		t.Errorf("expected scriptFile=scripts/interact.js, got %q", capturedScriptFile)
	}
}

func TestRunObserve_SendsScriptFile(t *testing.T) {
	dir := t.TempDir()
	store, _ := observation.NewStore(dir)
	scriptsDir := filepath.Join(dir, "scripts")
	mkdirAll(scriptsDir)
	snapshotJSON := `{"url":"https://example.com","title":"Test","timestamp":"2026-01-01T00:00:00Z","interactiveElements":[],"totalInteractiveElements":0,"visibleText":"hello","sections":[]}`
	writeTestFile(t, filepath.Join(scriptsDir, "observe.js"), `(function(){ return '`+snapshotJSON+`'; })()`)

	var capturedScriptFile string
	executeFileFn := func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		capturedScriptFile = scriptFile
		return true, snapshotJSON, "", nil
	}
	h := NewHandler(store, mockExecute(""), executeFileFn, mockScreenshot(), mockTabs(), scriptsDir, "")

	_, err := h.RunObserve("test", "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedScriptFile != "scripts/observe.js" {
		t.Errorf("expected scriptFile=scripts/observe.js, got %q", capturedScriptFile)
	}
}

func TestExecuteRaw_NoScriptFile(t *testing.T) {
	dir := t.TempDir()
	store, _ := observation.NewStore(dir)
	scriptsDir := filepath.Join(dir, "scripts")
	mkdirAll(scriptsDir)
	snapshotJSON := `{"url":"https://example.com","title":"Test","timestamp":"2026-01-01T00:00:00Z","interactiveElements":[],"totalInteractiveElements":0,"visibleText":"hello","sections":[]}`
	writeTestFile(t, filepath.Join(scriptsDir, "observe.js"), `(function(){ return '`+snapshotJSON+`'; })()`)

	// Track whether execute (code-only) was called for the raw code.
	executeCalled := false
	executeFn := func(code string) (bool, interface{}, string, error) {
		executeCalled = true
		return true, snapshotJSON, "", nil
	}

	// Track all scriptFile values passed to executeFile.
	var executeFileScriptFiles []string
	executeFileFn := func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		executeFileScriptFiles = append(executeFileScriptFiles, scriptFile)
		return true, snapshotJSON, "", nil
	}
	h := NewHandler(store, executeFn, executeFileFn, mockScreenshot(), mockTabs(), scriptsDir, "")

	_, _, _, _, err := h.ExecuteRaw("return 1+1;", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ExecuteRaw must use h.execute for the raw code (not h.executeFile).
	if !executeCalled {
		t.Error("expected execute (code-only) to be called for raw code")
	}

	// executeFile should only be called from RunObserve (auto-observe), with "scripts/observe.js".
	for _, sf := range executeFileScriptFiles {
		if sf != "scripts/observe.js" {
			t.Errorf("executeFile called with unexpected scriptFile %q (only scripts/observe.js expected from auto-observe)", sf)
		}
	}
}
