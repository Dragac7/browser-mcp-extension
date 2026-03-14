package e2e_test

import (
	"strings"
	"testing"
)

func TestWaitFor_TextAppears(t *testing.T) {
	navigateTo(t, "interactive.html")

	result := callTool(t, "browser_wait_for", map[string]interface{}{
		"text":    "Delayed text appeared",
		"timeout": 15,
	})
	parsed := parseToolResult(t, result)

	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", parsed["success"])
	}

	snap := snapshotPage(t)
	mustContainText(t, snap, "Delayed text appeared")
}

func TestWaitFor_TextTimeout(t *testing.T) {
	navigateTo(t, "basic.html")

	result := callTool(t, "browser_wait_for", map[string]interface{}{
		"text":    "Nonexistent text",
		"timeout": 3,
	})
	parsed := parseToolResult(t, result)

	// wait_for.js returns timeout via "return" (not throw), so success=true at the script level
	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true (wait_for.js uses return for timeouts), got ok=%v success=%v", ok, success)
	}

	data, ok := parsed["data"].(string)
	if !ok {
		t.Fatalf("expected data to be a string, got %T", parsed["data"])
	}
	if !strings.Contains(data, "Timeout") {
		t.Errorf("expected data to contain 'Timeout', got %q", data)
	}
}

func TestWaitFor_TextDisappears(t *testing.T) {
	navigateTo(t, "interactive.html")

	result := callTool(t, "browser_wait_for", map[string]interface{}{
		"textGone": "This will disappear",
		"timeout":  15,
	})
	parsed := parseToolResult(t, result)

	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", parsed["success"])
	}
}

func TestWaitFor_Duration(t *testing.T) {
	navigateTo(t, "basic.html")

	result := callTool(t, "browser_wait_for", map[string]interface{}{
		"time": 1,
	})
	parsed := parseToolResult(t, result)

	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", parsed["success"])
	}
}
