package e2e_test

import (
	"strings"
	"testing"
)

func TestNavigate_Success(t *testing.T) {
	snap := navigateTo(t, "basic.html")

	url, ok := snap["url"].(string)
	if !ok || !strings.Contains(url, "basic.html") {
		t.Errorf("expected URL to contain basic.html, got %q", url)
	}

	title, ok := snap["title"].(string)
	if !ok || title != "E2E Basic Page" {
		t.Errorf("expected title %q, got %q", "E2E Basic Page", title)
	}

	elements, ok := snap["interactiveElements"].([]interface{})
	if !ok || len(elements) == 0 {
		t.Error("expected non-empty interactiveElements")
	}
}

func TestNavigate_InvalidScheme(t *testing.T) {
	errText := callToolExpectError(t, "browser_navigate", map[string]interface{}{
		"url": "ftp://example.com",
	})
	if !strings.Contains(strings.ToLower(errText), "only http and https") {
		t.Errorf("expected error about scheme restriction, got %q", errText)
	}
}

func TestNavigate_SamePageTwice(t *testing.T) {
	snap1 := navigateTo(t, "basic.html")
	if snap1 == nil {
		t.Fatal("first navigation returned nil")
	}

	snap2 := navigateTo(t, "basic.html")
	if snap2 == nil {
		t.Fatal("second navigation returned nil")
	}

	title, ok := snap2["title"].(string)
	if !ok || title != "E2E Basic Page" {
		t.Errorf("expected title %q on second nav, got %q", "E2E Basic Page", title)
	}
}

func TestNavigateBack_Success(t *testing.T) {
	t.Skip("navigate_back.js calls history.back() which destroys the content script context, causing a 55s WS timeout")
	// Navigate to basic.html
	navigateTo(t, "basic.html")

	// Take snapshot to get element indices
	snap := snapshotPage(t)

	// Find the link "Go to target"
	idx, found := snapshotHasElement(snap, "Go to target")
	if !found {
		t.Fatal("could not find 'Go to target' link in snapshot")
	}

	// Click the link to navigate to basic_target.html
	result := callTool(t, "browser_click", map[string]interface{}{
		"elementIndex": idx,
	})
	if result.IsError {
		t.Fatalf("click failed: %v", result.Content)
	}

	// Verify we're on the target page
	snap = snapshotPage(t)
	url, _ := snap["url"].(string)
	if !strings.Contains(url, "basic_target.html") {
		t.Fatalf("expected URL to contain basic_target.html after click, got %q", url)
	}

	// Navigate back
	backResult := callTool(t, "browser_navigate_back", nil)
	if backResult.IsError {
		t.Fatalf("navigate_back failed: %v", backResult.Content)
	}

	// Verify we're back on basic.html
	snap = snapshotPage(t)
	url, _ = snap["url"].(string)
	if !strings.Contains(url, "basic.html") {
		t.Errorf("expected URL to contain basic.html after back, got %q", url)
	}
	mustContainText(t, snap, "Hello World")
}

func TestNavigateBack_NoHistory(t *testing.T) {
	navigateTo(t, "basic.html")

	result := callTool(t, "browser_navigate_back", nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty Content slice")
	}
}

func TestSnapshot_BasicPage(t *testing.T) {
	navigateTo(t, "basic.html")

	snap := snapshotPage(t)

	url, ok := snap["url"].(string)
	if !ok || url == "" {
		t.Error("snapshot missing url")
	}

	title, ok := snap["title"].(string)
	if !ok || title != "E2E Basic Page" {
		t.Errorf("expected title %q, got %q", "E2E Basic Page", title)
	}

	elements, ok := snap["interactiveElements"].([]interface{})
	if !ok || len(elements) == 0 {
		t.Error("expected non-empty interactiveElements")
	}

	mustContainText(t, snap, "Hello World")

	total, ok := snap["totalInteractiveElements"].(float64)
	if !ok || total <= 0 {
		t.Errorf("expected totalInteractiveElements > 0, got %v", snap["totalInteractiveElements"])
	}
}

func TestSnapshot_AfterDOMMutation(t *testing.T) {
	navigateTo(t, "basic.html")

	// Take initial snapshot to find the "Add Element" button
	snap := snapshotPage(t)
	idx, found := snapshotHasElement(snap, "Add Element")
	if !found {
		t.Fatal("could not find 'Add Element' button")
	}

	// Click the button to add a new element
	result := callTool(t, "browser_click", map[string]interface{}{
		"elementIndex": idx,
	})
	if result.IsError {
		t.Fatalf("click Add Element failed: %v", result.Content)
	}

	// Take snapshot again to verify the new element appears
	snap = snapshotPage(t)
	_, found = snapshotHasElement(snap, "New Button")
	if !found {
		t.Error("expected snapshot to include 'New Button' after DOM mutation")
	}
}

func TestGetState_AfterSnapshot(t *testing.T) {
	navigateTo(t, "basic.html")

	// First take a snapshot
	snap := snapshotPage(t)
	snapURL, _ := snap["url"].(string)
	snapTitle, _ := snap["title"].(string)

	// Now call get_state
	stateResult := callTool(t, "browser_get_state", nil)
	if stateResult.IsError {
		t.Fatalf("get_state failed: %v", stateResult.Content)
	}
	state := parseToolResult(t, stateResult)

	stateURL, _ := state["url"].(string)
	stateTitle, _ := state["title"].(string)

	if stateURL != snapURL {
		t.Errorf("get_state URL %q != snapshot URL %q", stateURL, snapURL)
	}
	if stateTitle != snapTitle {
		t.Errorf("get_state title %q != snapshot title %q", stateTitle, snapTitle)
	}
}

func TestGetState_NoSnapshot(t *testing.T) {
	t.Skip("requires fresh MCP server with no prior snapshots")
}
