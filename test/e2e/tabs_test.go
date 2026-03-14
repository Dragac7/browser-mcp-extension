package e2e_test

import (
	"strings"
	"testing"
)

func TestTabs_List(t *testing.T) {
	navigateTo(t, "tabs.html")

	result := callTool(t, "browser_tabs", map[string]interface{}{
		"action": "list",
	})
	parsed := parseToolResult(t, result)

	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", parsed["success"])
	}

	tabs, ok := parsed["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data to be an array, got %T", parsed["data"])
	}
	if len(tabs) < 1 {
		t.Fatal("expected at least 1 tab")
	}

	for i, tab := range tabs {
		tabMap, ok := tab.(map[string]interface{})
		if !ok {
			t.Fatalf("tab[%d] is not a map: %T", i, tab)
		}
		if _, ok := tabMap["url"].(string); !ok {
			t.Errorf("tab[%d] missing url string", i)
		}
		if _, ok := tabMap["title"].(string); !ok {
			t.Errorf("tab[%d] missing title string", i)
		}
	}
}

func TestTabs_Create(t *testing.T) {
	navigateTo(t, "tabs.html")

	// List tabs and note count
	listResult := callTool(t, "browser_tabs", map[string]interface{}{
		"action": "list",
	})
	listParsed := parseToolResult(t, listResult)
	tabs, ok := listParsed["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data to be an array, got %T", listParsed["data"])
	}
	initialCount := len(tabs)

	// Create a new tab
	createResult := callTool(t, "browser_tabs", map[string]interface{}{
		"action": "create",
		"url":    pageURL("basic.html"),
	})
	createParsed := parseToolResult(t, createResult)
	success, ok := createParsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true for create, got %v", createParsed["success"])
	}

	// List tabs again
	listResult2 := callTool(t, "browser_tabs", map[string]interface{}{
		"action": "list",
	})
	listParsed2 := parseToolResult(t, listResult2)
	tabs2, ok := listParsed2["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data to be an array, got %T", listParsed2["data"])
	}
	if len(tabs2) != initialCount+1 {
		t.Errorf("expected tab count to increase by 1: was %d, now %d", initialCount, len(tabs2))
	}

	// Cleanup: close the newly created tab
	t.Cleanup(func() {
		callTool(t, "browser_tabs", map[string]interface{}{
			"action": "close",
			"index":  len(tabs2) - 1,
		})
	})
}

func TestTabs_Select(t *testing.T) {
	navigateTo(t, "tabs.html")

	// Create a new tab to have at least 2
	callTool(t, "browser_tabs", map[string]interface{}{
		"action": "create",
		"url":    pageURL("basic_target.html"),
	})

	// List tabs to confirm 2+
	listResult := callTool(t, "browser_tabs", map[string]interface{}{
		"action": "list",
	})
	listParsed := parseToolResult(t, listResult)
	tabs, ok := listParsed["data"].([]interface{})
	if !ok || len(tabs) < 2 {
		t.Fatal("expected at least 2 tabs")
	}

	// Select tab at index 0
	selectResult := callTool(t, "browser_tabs", map[string]interface{}{
		"action": "select",
		"index":  0,
	})
	selectParsed := parseToolResult(t, selectResult)
	success, ok := selectParsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true for select, got %v", selectParsed["success"])
	}

	// Cleanup: close the extra tab
	t.Cleanup(func() {
		callTool(t, "browser_tabs", map[string]interface{}{
			"action": "close",
			"index":  len(tabs) - 1,
		})
	})
}

func TestTabs_Close(t *testing.T) {
	navigateTo(t, "tabs.html")

	// Create a new tab to close
	callTool(t, "browser_tabs", map[string]interface{}{
		"action": "create",
		"url":    pageURL("basic.html"),
	})

	// List tabs and note count
	listResult := callTool(t, "browser_tabs", map[string]interface{}{
		"action": "list",
	})
	listParsed := parseToolResult(t, listResult)
	tabs, ok := listParsed["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data to be an array, got %T", listParsed["data"])
	}
	countBefore := len(tabs)

	// Close the last tab
	closeResult := callTool(t, "browser_tabs", map[string]interface{}{
		"action": "close",
		"index":  countBefore - 1,
	})
	closeParsed := parseToolResult(t, closeResult)
	success, ok := closeParsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true for close, got %v", closeParsed["success"])
	}

	// List tabs again
	listResult2 := callTool(t, "browser_tabs", map[string]interface{}{
		"action": "list",
	})
	listParsed2 := parseToolResult(t, listResult2)
	tabs2, ok := listParsed2["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data to be an array, got %T", listParsed2["data"])
	}
	if len(tabs2) != countBefore-1 {
		t.Errorf("expected tab count to decrease by 1: was %d, now %d", countBefore, len(tabs2))
	}
}

func TestTabs_CreateInvalidScheme(t *testing.T) {
	errText := callToolExpectError(t, "browser_tabs", map[string]interface{}{
		"action": "create",
		"url":    "ftp://example.com",
	})
	if !strings.Contains(strings.ToLower(errText), "only http and https") {
		t.Errorf("expected error about scheme restriction, got %q", errText)
	}
}
