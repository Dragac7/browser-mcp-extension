package e2e_test

import (
	"strings"
	"testing"
)

func TestEvaluate_ReturnTitle(t *testing.T) {
	navigateTo(t, "evaluate.html")

	result := callTool(t, "browser_evaluate", map[string]interface{}{
		"code": "return document.title",
	})
	parsed := parseToolResult(t, result)

	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", parsed["success"])
	}

	data, ok := parsed["data"].(string)
	if !ok {
		t.Fatalf("expected data to be a string, got %T", parsed["data"])
	}
	if data != "E2E Evaluate Page" {
		t.Errorf("expected data %q, got %q", "E2E Evaluate Page", data)
	}
}

func TestEvaluate_DOMMutation(t *testing.T) {
	navigateTo(t, "evaluate.html")

	result := callTool(t, "browser_evaluate", map[string]interface{}{
		"code": "document.getElementById('result').textContent = 'done'; return 'ok'",
	})
	parsed := parseToolResult(t, result)

	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", parsed["success"])
	}

	snap := snapshotPage(t)
	mustContainText(t, snap, "done")
}

func TestEvaluate_SyntaxError(t *testing.T) {
	navigateTo(t, "evaluate.html")

	// Use null.toString() which throws a TypeError immediately (no 60s timeout).
	// Script errors via throw/exception return success=false in the JSON result
	// (NOT IsError=true at the MCP level), so we use callTool + parseToolResult.
	result := callTool(t, "browser_evaluate", map[string]interface{}{
		"code": "return null.toString()",
	})
	parsed := parseToolResult(t, result)

	success, ok := parsed["success"].(bool)
	if !ok || success {
		t.Errorf("expected success=false for runtime error, got ok=%v success=%v", ok, success)
	}
}

func TestEvaluate_ComplexReturn(t *testing.T) {
	navigateTo(t, "evaluate.html")

	result := callTool(t, "browser_evaluate", map[string]interface{}{
		"code": `return ({a: 1, b: "two"})`,
	})
	parsed := parseToolResult(t, result)

	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", parsed["success"])
	}

	data, ok := parsed["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", parsed["data"])
	}
	if data["a"] != float64(1) {
		t.Errorf("expected data.a = 1, got %v", data["a"])
	}
	if data["b"] != "two" {
		t.Errorf("expected data.b = 'two', got %v", data["b"])
	}
}

func TestEvaluate_WithParams(t *testing.T) {
	navigateTo(t, "evaluate.html")

	result := callTool(t, "browser_evaluate", map[string]interface{}{
		"code":   "return params.x + params.y",
		"params": map[string]interface{}{"x": 10, "y": 20},
	})
	parsed := parseToolResult(t, result)

	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", parsed["success"])
	}

	data, ok := parsed["data"].(float64)
	if !ok {
		t.Fatalf("expected data to be a number, got %T", parsed["data"])
	}
	if data != 30 {
		t.Errorf("expected data = 30, got %v", data)
	}
}

func TestEvaluate_EmptyCode(t *testing.T) {
	navigateTo(t, "evaluate.html")

	result := callTool(t, "browser_evaluate", map[string]interface{}{
		"code": "",
	})
	if result == nil {
		t.Fatal("expected non-nil result for empty code")
	}
}

func TestExecuteScript_Observe(t *testing.T) {
	navigateTo(t, "evaluate.html")

	result := callTool(t, "browser_execute_script", map[string]interface{}{
		"script": "observe.js",
	})
	parsed := parseToolResult(t, result)

	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", parsed["success"])
	}

	str, ok := parsed["data"].(string)
	if !ok {
		t.Fatalf("expected data to be a string, got %T", parsed["data"])
	}
	if !strings.Contains(str, "interactiveElements") {
		t.Errorf("expected data to contain 'interactiveElements', got %q", str)
	}

	snap, ok := parsed["snapshot"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected snapshot to be a map, got %T", parsed["snapshot"])
	}
	snapURL, ok := snap["url"].(string)
	if !ok || snapURL == "" {
		t.Error("expected snapshot url to be a non-empty string")
	}
	snapTitle, ok := snap["title"].(string)
	if !ok || snapTitle == "" {
		t.Error("expected snapshot title to be a non-empty string")
	}
}

func TestExecuteScript_NotFound(t *testing.T) {
	navigateTo(t, "evaluate.html")

	errText := callToolExpectError(t, "browser_execute_script", map[string]interface{}{
		"script": "nonexistent.js",
	})
	if !strings.Contains(strings.ToLower(errText), "script not found") {
		t.Errorf("expected error about script not found, got %q", errText)
	}
}

func TestExecuteScript_PathTraversal(t *testing.T) {
	navigateTo(t, "evaluate.html")

	errText := callToolExpectError(t, "browser_execute_script", map[string]interface{}{
		"script": "../../../etc/passwd",
	})
	if !strings.Contains(strings.ToLower(errText), "invalid script name") {
		t.Errorf("expected error about invalid script name, got %q", errText)
	}
}

func TestListScripts_ReturnsScripts(t *testing.T) {
	result := callTool(t, "browser_list_scripts", nil)
	parsed := parseToolResult(t, result)

	scripts, ok := parsed["scripts"].([]interface{})
	if !ok {
		t.Fatalf("expected scripts to be an array, got %T", parsed["scripts"])
	}

	// Collect script names
	scriptNames := make(map[string]bool)
	for _, s := range scripts {
		entry, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		name, ok := entry["name"].(string)
		if !ok {
			continue
		}
		scriptNames[name] = true
	}

	// Assert expected scripts are present
	expected := []string{
		"navigate.js", "navigate_back.js", "press_key.js", "scroll.js",
		"drag.js", "fill_form.js", "select_option.js", "wait_for.js", "query_click.js",
	}
	for _, name := range expected {
		if !scriptNames[name] {
			t.Errorf("expected script %q in list, not found", name)
		}
	}

	// Assert excluded scripts are NOT present
	excluded := []string{"observe.js", "interact.js"}
	for _, name := range excluded {
		if scriptNames[name] {
			t.Errorf("script %q should be excluded from list", name)
		}
	}
}
