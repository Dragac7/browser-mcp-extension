package api

import (
	"encoding/json"
	"fmt"
	"github.com/paoloandrisani/browser-mcp-extension/internal/observation"
	"log"
	"net/http"
	"net/url"
)

// ════════════════════════════════════════════════════════════════════════
// Browser-specific exported methods (used by MCP layer)
// ════════════════════════════════════════════════════════════════════════

// BrowserScreenshot takes a screenshot of the current tab.
func (h *Handler) BrowserScreenshot() (dataURL string, err error) {
	return h.screenshot()
}

// BrowserTabs manages browser tabs.
func (h *Handler) BrowserTabs(action string, index *int, url string) (bool, interface{}, string, error) {
	return h.tabs(action, index, url)
}

// ════════════════════════════════════════════════════════════════════════
// HTTP handlers
// ════════════════════════════════════════════════════════════════════════

func (h *Handler) handleState(w http.ResponseWriter, r *http.Request) {
	snap, err := h.GetState()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if snap == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "no snapshot yet"})
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (h *Handler) handleObserve(w http.ResponseWriter, r *http.Request) {
	snap, err := h.RunObserve("manual", "", true)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (h *Handler) handleExecute(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Script string                 `json:"script"`
		Params map[string]interface{} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	ok, dataStr, errMsg, snap, err := h.ExecuteScriptAndObserve(req.Script, req.Params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  ok,
		"data":     dataStr,
		"error":    errMsg,
		"snapshot": snap,
	})
}

func (h *Handler) handleExecuteRaw(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code   string                 `json:"code"`
		Params map[string]interface{} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	ok, data, errMsg, snap, err := h.ExecuteRaw(req.Code, req.Params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  ok,
		"data":     data,
		"error":    errMsg,
		"snapshot": snap,
	})
}

func (h *Handler) handleScripts(w http.ResponseWriter, r *http.Request) {
	scripts, err := h.ListScripts()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"scripts": scripts})
}

func (h *Handler) handleScreenshot(w http.ResponseWriter, r *http.Request) {
	url, err := h.screenshot()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"dataURL": url})
}

func (h *Handler) handleTabs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
		Index  *int   `json:"index"`
		URL    string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.Action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action is required"})
		return
	}
	// Validate URL scheme for the create action.
	if req.Action == "create" && req.URL != "" {
		if err := validateHTTPURL(req.URL); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	ok, data, errMsg, err := h.tabs(req.Action, req.Index, req.URL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": ok,
		"data":    data,
		"error":   errMsg,
	})
}

func validateHTTPURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("invalid URL %q: only http and https schemes are supported", rawURL)
	}
	return nil
}

// ════════════════════════════════════════════════════════════════════════
// Exported interaction helpers (used by MCP tools layer)
// ════════════════════════════════════════════════════════════════════════

// BrowserInteract runs interact.js with the given action and params, then observes.
func (h *Handler) BrowserInteract(action string, extraParams map[string]interface{}) (bool, string, string, *observation.Snapshot, error) {
	params := map[string]interface{}{"action": action}
	for k, v := range extraParams {
		params[k] = v
	}
	return h.ExecuteScriptAndObserve("interact.js", params)
}

// Snapshot captures a fresh page snapshot and returns it.
func (h *Handler) Snapshot() (*observation.Snapshot, error) {
	return h.RunObserve("snapshot", "", true)
}

// NavigateBack calls navigate_back.js and auto-observes the resulting page.
func (h *Handler) NavigateBack() (string, *observation.Snapshot, error) {
	ok, data, errMsg, err := h.ExecuteScript("navigate_back.js", nil)
	if err != nil {
		return "", nil, err
	}
	if !ok {
		return "", nil, fmt.Errorf("navigate_back failed: %s", errMsg)
	}
	snap, observeErr := h.RunObserve("navigate_back", data, true)
	if observeErr != nil {
		log.Printf("[API] ⚠ observe after navigate_back failed: %v", observeErr)
	}
	return data, snap, nil
}
