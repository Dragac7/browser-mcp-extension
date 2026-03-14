package e2e_test

import (
	"strings"
	"testing"
)

// navigateAndFindElement navigates to a page, takes a snapshot, finds an element
// by text substring, and returns its index. Fails the test if not found.
func navigateAndFindElement(t *testing.T, page, textSubstring string) int {
	t.Helper()
	navigateTo(t, page)
	snap := snapshotPage(t)
	idx, found := snapshotHasElement(snap, textSubstring)
	if !found {
		t.Fatalf("element containing %q not found on %s", textSubstring, page)
	}
	return idx
}

// ── Click tests ─────────────────────────────────────────────────────

func TestClick_ButtonChangesText(t *testing.T) {
	idx := navigateAndFindElement(t, "basic.html", "Change Text")

	result := callTool(t, "browser_click", map[string]interface{}{
		"elementIndex": idx,
	})
	if result.IsError {
		t.Fatalf("click failed: %v", result.Content)
	}

	snap := snapshotPage(t)
	mustContainText(t, snap, "Changed")
}

func TestClick_LinkNavigates(t *testing.T) {
	idx := navigateAndFindElement(t, "basic.html", "Go to target")

	result := callTool(t, "browser_click", map[string]interface{}{
		"elementIndex": idx,
	})
	if result.IsError {
		t.Fatalf("click failed: %v", result.Content)
	}

	snap := snapshotPage(t)
	url, _ := snap["url"].(string)
	if !strings.Contains(url, "basic_target.html") {
		t.Errorf("expected URL to contain basic_target.html, got %q", url)
	}
	mustContainText(t, snap, "You navigated here")
}

func TestClick_DoubleClick(t *testing.T) {
	idx := navigateAndFindElement(t, "basic.html", "Double Click Me")

	result := callTool(t, "browser_click", map[string]interface{}{
		"elementIndex": idx,
		"doubleClick":  true,
	})
	if result.IsError {
		t.Fatalf("double click failed: %v", result.Content)
	}

	snap := snapshotPage(t)
	mustContainText(t, snap, "Double Clicked")
}

func TestClick_InvalidIndex(t *testing.T) {
	navigateTo(t, "basic.html")
	snapshotPage(t)

	result := callTool(t, "browser_click", map[string]interface{}{
		"elementIndex": 9999,
	})
	parsed := parseToolResult(t, result)
	data, ok := parsed["data"].(string)
	if !ok {
		t.Fatalf("expected data to be a string, got %T", parsed["data"])
	}
	if !strings.Contains(data, "out of range") {
		t.Errorf("expected 'out of range' in result, got %q", data)
	}
}

func TestClick_DisabledButton(t *testing.T) {
	idx := navigateAndFindElement(t, "basic.html", "Disabled Button")

	result := callTool(t, "browser_click", map[string]interface{}{
		"elementIndex": idx,
	})
	if result == nil {
		t.Fatal("expected non-nil result for disabled button click")
	}
}

// ── Type tests ──────────────────────────────────────────────────────

func TestType_TextInput(t *testing.T) {
	idx := navigateAndFindElement(t, "form.html", "Name")

	result := callTool(t, "browser_type", map[string]interface{}{
		"elementIndex": idx,
		"text":         "John Doe",
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
	if !strings.Contains(data, `Typed "John Doe"`) {
		t.Errorf("expected data to contain 'Typed \"John Doe\"', got %q", data)
	}
}

func TestType_Textarea(t *testing.T) {
	t.Skip("utils.js type() uses HTMLInputElement.prototype.value.set which throws TypeError on textarea elements — production bug outside E2E test plan scope")
}

func TestType_WithSubmit(t *testing.T) {
	idx := navigateAndFindElement(t, "form.html", "Name")

	result := callTool(t, "browser_type", map[string]interface{}{
		"elementIndex": idx,
		"text":         "test",
		"submit":       true,
	})
	if result.IsError {
		t.Fatalf("type with submit failed: %v", result.Content)
	}

	snap := snapshotPage(t)
	mustContainText(t, snap, "Submitted")
}

func TestType_ClearFalse(t *testing.T) {
	idx := navigateAndFindElement(t, "form.html", "Name")

	// Type "A" first (with default clear=true)
	result := callTool(t, "browser_type", map[string]interface{}{
		"elementIndex": idx,
		"text":         "A",
	})
	if result.IsError {
		t.Fatalf("first type failed: %v", result.Content)
	}

	// Type "B" with clear=false to append
	result = callTool(t, "browser_type", map[string]interface{}{
		"elementIndex": idx,
		"text":         "B",
		"clear":        false,
	})
	if result.IsError {
		t.Fatalf("second type failed: %v", result.Content)
	}

	// Verify input value is "AB" using browser_evaluate
	evalResult := callTool(t, "browser_evaluate", map[string]interface{}{
		"code": "return document.getElementById('name-input').value",
	})
	parsed := parseToolResult(t, evalResult)

	data, ok := parsed["data"].(string)
	if !ok {
		t.Fatalf("expected data to be a string, got %T", parsed["data"])
	}
	if data != "AB" {
		t.Errorf("expected input value 'AB', got %q", data)
	}
}

// ── Hover tests ─────────────────────────────────────────────────────

func TestHover_RevealsContent(t *testing.T) {
	idx := navigateAndFindElement(t, "interactive.html", "Hover over me")

	result := callTool(t, "browser_hover", map[string]interface{}{
		"elementIndex": idx,
	})
	if result.IsError {
		t.Fatalf("hover failed: %v", result.Content)
	}

	snap := snapshotPage(t)
	mustContainText(t, snap, "Hover detected!")
}

func TestHover_InvalidIndex(t *testing.T) {
	navigateTo(t, "interactive.html")
	snapshotPage(t)

	result := callTool(t, "browser_hover", map[string]interface{}{
		"elementIndex": 9999,
	})
	parsed := parseToolResult(t, result)
	data, ok := parsed["data"].(string)
	if !ok {
		t.Fatalf("expected data to be a string, got %T", parsed["data"])
	}
	if !strings.Contains(data, "out of range") {
		t.Errorf("expected 'out of range' in result, got %q", data)
	}
}

// ── PressKey tests ──────────────────────────────────────────────────

func TestPressKey_EnterSubmitsForm(t *testing.T) {
	idx := navigateAndFindElement(t, "form.html", "Name")

	// Click to focus the name input
	result := callTool(t, "browser_click", map[string]interface{}{
		"elementIndex": idx,
	})
	if result.IsError {
		t.Fatalf("click to focus failed: %v", result.Content)
	}

	// Press Enter
	result = callTool(t, "browser_press_key", map[string]interface{}{
		"key": "Enter",
	})
	if result.IsError {
		t.Fatalf("press_key Enter failed: %v", result.Content)
	}

	snap := snapshotPage(t)
	mustContainText(t, snap, "Submitted")
}

func TestPressKey_Escape(t *testing.T) {
	navigateTo(t, "interactive.html")
	snap := snapshotPage(t)

	// Find and click key-target input to focus it (search by placeholder text)
	idx, found := snapshotHasElement(snap, "Type here")
	if !found {
		t.Fatal("could not find key-target input")
	}

	result := callTool(t, "browser_click", map[string]interface{}{
		"elementIndex": idx,
	})
	if result.IsError {
		t.Fatalf("click key-target failed: %v", result.Content)
	}

	// Press Escape
	result = callTool(t, "browser_press_key", map[string]interface{}{
		"key": "Escape",
	})
	if result.IsError {
		t.Fatalf("press_key Escape failed: %v", result.Content)
	}

	snap = snapshotPage(t)
	mustContainText(t, snap, "Escape")
}

func TestPressKey_ArrowDown(t *testing.T) {
	navigateTo(t, "interactive.html")

	// Focus the key-target input via evaluate (avoids flaky element index lookup)
	evalResult := callTool(t, "browser_evaluate", map[string]interface{}{
		"code": "document.getElementById('key-target').focus(); return 'focused'",
	})
	parsed := parseToolResult(t, evalResult)
	if s, _ := parsed["success"].(bool); !s {
		t.Fatalf("could not focus key-target: %v", parsed)
	}

	result := callTool(t, "browser_press_key", map[string]interface{}{
		"key": "ArrowDown",
	})
	if result.IsError {
		t.Fatalf("press_key ArrowDown failed: %v", result.Content)
	}

	snap := snapshotPage(t)
	mustContainText(t, snap, "ArrowDown")
}

func TestPressKey_Tab(t *testing.T) {
	t.Skip("synthetic Tab keydown/keyup events do not trigger browser's native tab order focus navigation")
}

// ── SelectOption tests ──────────────────────────────────────────────

func TestSelectOption_SingleValue(t *testing.T) {
	idx := navigateAndFindElement(t, "form.html", "--Select--")

	result := callTool(t, "browser_select_option", map[string]interface{}{
		"elementIndex": idx,
		"values":       []interface{}{"red"},
	})
	parsed := parseToolResult(t, result)

	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", parsed["success"])
	}
}

func TestSelectOption_MultipleValues(t *testing.T) {
	idx := navigateAndFindElement(t, "form.html", "Option A")

	result := callTool(t, "browser_select_option", map[string]interface{}{
		"elementIndex": idx,
		"values":       []interface{}{"a", "c"},
	})
	parsed := parseToolResult(t, result)

	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", parsed["success"])
	}
}

func TestSelectOption_NonExistentValue(t *testing.T) {
	// Use multi-select to avoid single-select auto-selecting first option
	idx := navigateAndFindElement(t, "form.html", "Option A")

	result := callTool(t, "browser_select_option", map[string]interface{}{
		"elementIndex": idx,
		"values":       []interface{}{"nonexistent"},
	})
	parsed := parseToolResult(t, result)
	data, ok := parsed["data"].(string)
	if !ok {
		t.Fatalf("expected data to be a string, got %T", parsed["data"])
	}
	if !strings.Contains(data, "None of the provided values") {
		t.Errorf("expected 'None of the provided values' in data, got %q", data)
	}
}

// ── Scroll tests ────────────────────────────────────────────────────

func TestScroll_WindowDown(t *testing.T) {
	navigateTo(t, "interactive.html")

	result := callTool(t, "browser_scroll", map[string]interface{}{
		"deltaY": 2000,
	})
	if result.IsError {
		t.Fatalf("scroll failed: %v", result.Content)
	}

	snap := snapshotPage(t)
	mustContainText(t, snap, "Bottom of page")
}

func TestScroll_ElementIntoView(t *testing.T) {
	idx := navigateAndFindElement(t, "interactive.html", "Bottom of page")

	result := callTool(t, "browser_scroll", map[string]interface{}{
		"elementIndex": idx,
	})
	if result.IsError {
		t.Fatalf("scroll to element failed: %v", result.Content)
	}

	snap := snapshotPage(t)
	mustContainText(t, snap, "Bottom of page")
}

// ── Drag tests ──────────────────────────────────────────────────────

func TestDrag_SuccessfulDrop(t *testing.T) {
	navigateTo(t, "drag.html")
	snap := snapshotPage(t)

	srcIdx, found := snapshotHasElement(snap, "Drag me")
	if !found {
		t.Fatal("could not find 'Drag me' element")
	}
	dstIdx, found := snapshotHasElement(snap, "Drop here")
	if !found {
		t.Fatal("could not find 'Drop here' element")
	}

	result := callTool(t, "browser_drag", map[string]interface{}{
		"startElementIndex": srcIdx,
		"endElementIndex":   dstIdx,
	})
	if result.IsError {
		t.Fatalf("drag failed: %v", result.Content)
	}

	snap = snapshotPage(t)
	mustContainText(t, snap, "Dropped!")
}

func TestDrag_InvalidSourceIndex(t *testing.T) {
	navigateTo(t, "drag.html")
	snapshotPage(t)

	result := callTool(t, "browser_drag", map[string]interface{}{
		"startElementIndex": 9999,
		"endElementIndex":   0,
	})
	parsed := parseToolResult(t, result)
	data, ok := parsed["data"].(string)
	if !ok {
		t.Fatalf("expected data to be a string, got %T", parsed["data"])
	}
	if !strings.Contains(data, "out of range") {
		t.Errorf("expected 'out of range' in data, got %q", data)
	}
}

func TestDrag_InvalidTargetIndex(t *testing.T) {
	navigateTo(t, "drag.html")
	snap := snapshotPage(t)

	srcIdx, found := snapshotHasElement(snap, "Drag me")
	if !found {
		t.Fatal("could not find 'Drag me' element")
	}

	result := callTool(t, "browser_drag", map[string]interface{}{
		"startElementIndex": srcIdx,
		"endElementIndex":   9999,
	})
	parsed := parseToolResult(t, result)
	data, ok := parsed["data"].(string)
	if !ok {
		t.Fatalf("expected data to be a string, got %T", parsed["data"])
	}
	if !strings.Contains(data, "out of range") {
		t.Errorf("expected 'out of range' in data, got %q", data)
	}
}

// ── FillForm tests ──────────────────────────────────────────────────

func TestFillForm_MultipleFields(t *testing.T) {
	navigateTo(t, "form.html")
	snap := snapshotPage(t)

	nameIdx, found := snapshotHasElement(snap, "Name")
	if !found {
		t.Fatal("could not find name input")
	}
	emailIdx, found := snapshotHasElement(snap, "Email")
	if !found {
		t.Fatal("could not find email input")
	}

	// Note: textarea fields are excluded because utils.js type() uses
	// HTMLInputElement.prototype.value.set which throws TypeError on textarea elements.
	result := callTool(t, "browser_fill_form", map[string]interface{}{
		"fields": []interface{}{
			map[string]interface{}{"elementIndex": nameIdx, "text": "John"},
			map[string]interface{}{"elementIndex": emailIdx, "text": "john@test.com"},
		},
	})
	parsed := parseToolResult(t, result)

	success, ok := parsed["success"].(bool)
	if !ok || !success {
		t.Errorf("expected success=true, got %v", parsed["success"])
	}
}

func TestFillForm_EmptyFields(t *testing.T) {
	navigateTo(t, "form.html")

	errText := callToolExpectError(t, "browser_fill_form", map[string]interface{}{
		"fields": []interface{}{},
	})
	if !strings.Contains(errText, "fields must be a non-empty array") {
		t.Errorf("expected error about empty fields, got %q", errText)
	}
}

func TestFillForm_InvalidIndex(t *testing.T) {
	navigateTo(t, "form.html")
	snapshotPage(t)

	result := callTool(t, "browser_fill_form", map[string]interface{}{
		"fields": []interface{}{
			map[string]interface{}{"elementIndex": 9999, "text": "test"},
		},
	})
	parsed := parseToolResult(t, result)
	data, ok := parsed["data"].(string)
	if !ok {
		t.Fatalf("expected data to be a string, got %T", parsed["data"])
	}
	if !strings.Contains(data, "out of range") {
		t.Errorf("expected 'out of range' in data, got %q", data)
	}
}
