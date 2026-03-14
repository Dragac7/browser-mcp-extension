package api

import (
	"encoding/json"
	"fmt"
	"github.com/paoloandrisani/browser-mcp-extension/internal/observation"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Handler holds the HTTP API state for the browser module.
type Handler struct {
	store       *observation.Store
	execute     ExecuteFn
	executeFile ExecuteFileFn
	screenshot  ScreenshotFn
	tabs        TabsFn
	scriptsDir  string
	libCode     string
}

// NewHandler creates a new API handler.
func NewHandler(
	store *observation.Store,
	executeFn ExecuteFn,
	executeFileFn ExecuteFileFn,
	screenshotFn ScreenshotFn,
	tabsFn TabsFn,
	scriptsDir string,
	libCode string,
) *Handler {
	return &Handler{
		store:       store,
		execute:     executeFn,
		executeFile: executeFileFn,
		screenshot:  screenshotFn,
		tabs:        tabsFn,
		scriptsDir:  scriptsDir,
		libCode:     libCode,
	}
}

// RegisterRoutes registers all HTTP API routes.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/state", h.handleState)
	mux.HandleFunc("/api/observe", h.handleObserve)
	mux.HandleFunc("/api/execute", h.handleExecute)
	mux.HandleFunc("/api/execute-raw", h.handleExecuteRaw)
	mux.HandleFunc("/api/scripts", h.handleScripts)
	mux.HandleFunc("/api/screenshot", h.handleScreenshot)
	mux.HandleFunc("/api/tabs", h.handleTabs)
}

// ════════════════════════════════════════════════════════════════════════
// Script execution helpers
// ════════════════════════════════════════════════════════════════════════

// ExecuteScript loads a JS script by name, wraps it, executes it, returns result.
// scriptName must resolve to a path within scriptsDir (path traversal is rejected).
func (h *Handler) ExecuteScript(scriptName string, params map[string]interface{}) (success bool, data string, errMsg string, err error) {
	scriptPath := filepath.Join(h.scriptsDir, scriptName)
	// Guard against path traversal (e.g. "../../../etc/passwd").
	cleanDir := filepath.Clean(h.scriptsDir) + string(os.PathSeparator)
	cleanPath := filepath.Clean(scriptPath)
	if !strings.HasPrefix(cleanPath+string(os.PathSeparator), cleanDir) {
		return false, "", "", fmt.Errorf("invalid script name: %s", scriptName)
	}
	scriptData, err := os.ReadFile(scriptPath)
	if err != nil {
		return false, "", "", fmt.Errorf("script not found: %s", scriptName)
	}
	code, err := WrapCode(string(scriptData), h.libCode, params)
	if err != nil {
		return false, "", "", fmt.Errorf("wrap error: %w", err)
	}
	scriptFile := "scripts/" + scriptName
	ok, rawData, eMsg, execErr := h.executeFile(scriptFile, code, params)
	if execErr != nil {
		return false, "", "", execErr
	}
	return ok, fmt.Sprintf("%v", rawData), eMsg, nil
}

// ExecuteScriptAndObserve runs a script and auto-observes afterward.
func (h *Handler) ExecuteScriptAndObserve(scriptName string, params map[string]interface{}) (success bool, data string, errMsg string, snap *observation.Snapshot, err error) {
	ok, dataStr, eMsg, err := h.ExecuteScript(scriptName, params)
	if err != nil {
		return false, "", "", nil, err
	}
	snap, observeErr := h.RunObserve(scriptName, dataStr, ok)
	if observeErr != nil {
		log.Printf("[API] ⚠ observe after %s failed: %v", scriptName, observeErr)
	}
	return ok, dataStr, eMsg, snap, nil
}

// NavigateTo validates and navigates to a URL, then waits for the page to load.
// Only http and https schemes are accepted.
func (h *Handler) NavigateTo(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("invalid URL %q: only http and https schemes are supported", rawURL)
	}
	ok, _, errMsg, execErr := h.ExecuteScript("navigate.js", map[string]interface{}{"url": rawURL})
	if execErr != nil {
		return execErr
	}
	if !ok {
		return fmt.Errorf("navigation failed: %s", errMsg)
	}
	time.Sleep(3 * time.Second)
	return nil
}

// GetState returns the latest page snapshot (memory then disk).
func (h *Handler) GetState() (*observation.Snapshot, error) {
	snap := h.store.Latest()
	if snap == nil {
		var err error
		snap, err = h.store.LatestFromDisk()
		if err != nil {
			return nil, err
		}
	}
	return snap, nil
}

// ListScripts returns all available automation scripts.
func (h *Handler) ListScripts() ([]ScriptEntry, error) {
	var scripts []ScriptEntry
	err := filepath.Walk(h.scriptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".js") {
			return nil
		}
		if info.Name() == "observe.js" || info.Name() == "interact.js" {
			return nil
		}
		rel, _ := filepath.Rel(h.scriptsDir, path)
		scripts = append(scripts, ScriptEntry{Name: rel, Path: rel})
		return nil
	})
	return scripts, err
}

// ExecuteRaw wraps arbitrary JS code, executes it, and auto-observes.
func (h *Handler) ExecuteRaw(code string, params map[string]interface{}) (success bool, data interface{}, errMsg string, snap *observation.Snapshot, err error) {
	wrapped, err := WrapCode(code, h.libCode, params)
	if err != nil {
		return false, nil, "", nil, fmt.Errorf("wrap error: %w", err)
	}
	success, data, errMsg, err = h.execute(wrapped)
	if err != nil {
		return false, nil, "", nil, fmt.Errorf("execution error: %w", err)
	}
	snap, observeErr := h.RunObserve("raw", fmt.Sprintf("%v", data), success)
	if observeErr != nil {
		log.Printf("[API] ⚠ observe after execute-raw failed: %v", observeErr)
	}
	return success, data, errMsg, snap, nil
}

// RunObserve executes observe.js and saves a page snapshot.
func (h *Handler) RunObserve(action, actionResult string, actionSuccess bool) (*observation.Snapshot, error) {
	observePath := filepath.Join(h.scriptsDir, "observe.js")
	observeData, err := os.ReadFile(observePath)
	if err != nil {
		return nil, fmt.Errorf("read observe.js: %w", err)
	}
	success, data, errMsg, err := h.executeFile("scripts/observe.js", string(observeData), nil)
	if err != nil {
		return nil, fmt.Errorf("execute observe.js: %w", err)
	}
	if !success {
		return nil, fmt.Errorf("observe.js failed: %s", errMsg)
	}
	dataStr, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("observe.js returned non-string: %T", data)
	}
	snap, err := h.store.Save(dataStr, action, actionResult, actionSuccess)
	if err != nil {
		return nil, fmt.Errorf("save snapshot: %w", err)
	}
	return snap, nil
}

// WrapCode wraps JS code with the utils library and optional params.
func WrapCode(code, libCode string, params map[string]interface{}) (string, error) {
	if len(params) > 0 {
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return "", fmt.Errorf("marshal params: %w", err)
		}
		return fmt.Sprintf("(async function(params) {\n%s\n\n%s\n})(%s)", libCode, code, string(paramsJSON)), nil
	}
	return fmt.Sprintf("(async function() {\n%s\n\n%s\n})()", libCode, code), nil
}

// ════════════════════════════════════════════════════════════════════════
// Internal HTTP helpers
// ════════════════════════════════════════════════════════════════════════

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
