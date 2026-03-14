package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/paoloandrisani/browser-mcp-extension/internal/api"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerTools(s *server.MCPServer, h *api.Handler) {
	// ── Navigation ──────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("browser_navigate",
		mcp.WithDescription("Navigate to a URL. Waits for page load and captures a snapshot automatically."),
		mcp.WithString("url", mcp.Required(), mcp.Description("Full URL to navigate to (e.g. https://example.com)")),
	), handleNavigate(h))

	s.AddTool(mcp.NewTool("browser_navigate_back",
		mcp.WithDescription("Go back one step in browser history."),
	), handleNavigateBack(h))

	// ── Observation ─────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("browser_snapshot",
		mcp.WithDescription("Capture a structured snapshot of the current page. Returns interactive elements with numeric indices, visible text, and sections. Always call this before interacting with elements."),
	), handleSnapshot(h))

	s.AddTool(mcp.NewTool("browser_get_state",
		mcp.WithDescription("Return the latest cached page snapshot without re-observing. Faster than browser_snapshot but may be stale."),
	), handleGetState(h))

	// ── Interaction ─────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("browser_click",
		mcp.WithDescription("Click an element by its index from the last browser_snapshot. Supports single and double click."),
		mcp.WithNumber("elementIndex", mcp.Required(), mcp.Description("Element index from the last browser_snapshot")),
		mcp.WithBoolean("doubleClick", mcp.Description("Perform a double-click instead of a single click (default: false)")),
	), handleClick(h))

	s.AddTool(mcp.NewTool("browser_type",
		mcp.WithDescription("Type text into an element by its index from the last browser_snapshot."),
		mcp.WithNumber("elementIndex", mcp.Required(), mcp.Description("Element index from the last browser_snapshot")),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to type")),
		mcp.WithBoolean("submit", mcp.Description("Press Enter after typing (default: false)")),
		mcp.WithBoolean("clear", mcp.Description("Clear existing value before typing (default: true)")),
	), handleType(h))

	s.AddTool(mcp.NewTool("browser_hover",
		mcp.WithDescription("Hover the mouse over an element by its index from the last browser_snapshot."),
		mcp.WithNumber("elementIndex", mcp.Required(), mcp.Description("Element index from the last browser_snapshot")),
	), handleHover(h))

	s.AddTool(mcp.NewTool("browser_press_key",
		mcp.WithDescription("Dispatch a keyboard key event on the currently focused element. Use standard key names: Enter, Escape, ArrowDown, Tab, etc."),
		mcp.WithString("key", mcp.Required(), mcp.Description("Key name per KeyboardEvent spec (e.g. Enter, Escape, ArrowDown, a)")),
	), handlePressKey(h))

	s.AddTool(mcp.NewTool("browser_select_option",
		mcp.WithDescription("Select one or more options in a <select> element by its index from the last browser_snapshot."),
		mcp.WithNumber("elementIndex", mcp.Required(), mcp.Description("Element index of the <select> from the last browser_snapshot")),
		mcp.WithArray("values", mcp.Required(), mcp.Description("Array of option values to select")),
	), handleSelectOption(h))

	s.AddTool(mcp.NewTool("browser_scroll",
		mcp.WithDescription("Scroll the page window or scroll an element into view by index."),
		mcp.WithNumber("elementIndex", mcp.Description("Element index to scroll into view (omit to scroll the window)")),
		mcp.WithNumber("deltaX", mcp.Description("Horizontal scroll delta in pixels (default: 0)")),
		mcp.WithNumber("deltaY", mcp.Description("Vertical scroll delta in pixels (default: 300)")),
	), handleScroll(h))

	s.AddTool(mcp.NewTool("browser_drag",
		mcp.WithDescription("Drag from a source element to a target element using their snapshot indices."),
		mcp.WithNumber("startElementIndex", mcp.Required(), mcp.Description("Source element index from the last browser_snapshot")),
		mcp.WithNumber("endElementIndex", mcp.Required(), mcp.Description("Target element index from the last browser_snapshot")),
	), handleDrag(h))

	s.AddTool(mcp.NewTool("browser_fill_form",
		mcp.WithDescription("Fill multiple form fields in a single call. Each field specifies an element index and text."),
		mcp.WithArray("fields", mcp.Required(), mcp.Description("Array of {elementIndex: number, text: string, clear?: boolean}")),
	), handleFillForm(h))

	s.AddTool(mcp.NewTool("browser_wait_for",
		mcp.WithDescription("Wait for a fixed duration, for text to appear on the page, or for text to disappear."),
		mcp.WithNumber("time", mcp.Description("Seconds to wait unconditionally")),
		mcp.WithString("text", mcp.Description("Text to wait for to appear on the page")),
		mcp.WithString("textGone", mcp.Description("Text to wait for to disappear from the page")),
		mcp.WithNumber("timeout", mcp.Description("Max seconds to poll when waiting for text (default: 10)")),
	), handleWaitFor(h))

	s.AddTool(mcp.NewTool("browser_evaluate",
		mcp.WithDescription("Execute arbitrary JavaScript in the page context. Has access to the shared utils library (wait, randomDelay, type, click, hover). Returns the result and a page snapshot."),
		mcp.WithString("code", mcp.Required(), mcp.Description("JavaScript code to evaluate in the page context")),
		mcp.WithObject("params", mcp.Description("Optional key-value parameters accessible as `params` in the code")),
	), handleEvaluate(h))

	// ── Extension-enhanced ──────────────────────────────────────────────
	s.AddTool(mcp.NewTool("browser_take_screenshot",
		mcp.WithDescription("Capture a screenshot of the current tab. Returns a PNG image as a base64 data URL."),
	), handleTakeScreenshot(h))

	s.AddTool(mcp.NewTool("browser_tabs",
		mcp.WithDescription("Manage browser tabs: list all tabs, create a new tab, close a tab, or switch to a tab."),
		mcp.WithString("action", mcp.Required(), mcp.Description("Operation: list | create | close | select")),
		mcp.WithNumber("index", mcp.Description("Tab index (required for close/select)")),
		mcp.WithString("url", mcp.Description("URL for the new tab (used with action=create)")),
	), handleTabs(h))

	// ── Script execution ────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("browser_execute_script",
		mcp.WithDescription("Execute a named JavaScript script from the scripts directory. Use browser_list_scripts to see available scripts."),
		mcp.WithString("script", mcp.Required(), mcp.Description("Script filename relative to the scripts directory (e.g. navigate.js)")),
		mcp.WithObject("params", mcp.Description("Optional key-value parameters to pass to the script")),
	), handleExecuteScript(h))

	s.AddTool(mcp.NewTool("browser_list_scripts",
		mcp.WithDescription("List all available JavaScript automation scripts that can be run with browser_execute_script."),
	), handleListScripts(h))
}

// ════════════════════════════════════════════════════════════════════════
// Tool handler implementations
// ════════════════════════════════════════════════════════════════════════

// toolResult marshals a value to indented JSON and returns it as a text tool result.
func toolResult(v interface{}) *mcp.CallToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err))
	}
	return mcp.NewToolResultText(string(data))
}

func handleNavigate(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rawURL, err := req.RequireString("url")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := h.NavigateTo(rawURL); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("navigation failed: %v", err)), nil
		}
		snap, err := h.RunObserve("navigate", rawURL, true)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Navigated to %s (observe failed: %v)", rawURL, err)), nil
		}
		return toolResult(snap), nil
	}
}

func handleNavigateBack(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, snap, err := h.NavigateBack()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("navigate back failed: %v", err)), nil
		}
		if snap != nil {
			return toolResult(map[string]interface{}{"data": result, "snapshot": snap}), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}

func handleSnapshot(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		snap, err := h.Snapshot()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("snapshot failed: %v", err)), nil
		}
		return toolResult(snap), nil
	}
}

func handleGetState(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		snap, err := h.GetState()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get state failed: %v", err)), nil
		}
		if snap == nil {
			return mcp.NewToolResultText(`{"status":"no snapshot yet","hint":"call browser_snapshot first"}`), nil
		}
		return toolResult(snap), nil
	}
}

func handleClick(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		idxF, ok := args["elementIndex"].(float64)
		if !ok {
			return mcp.NewToolResultError("elementIndex is required"), nil
		}
		params := map[string]interface{}{"elementIndex": int(idxF)}
		if dc, ok := args["doubleClick"].(bool); ok && dc {
			params["action"] = "double_click"
		}
		ok2, data, errMsg, snap, err := h.BrowserInteract("click", params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("click failed: %v", err)), nil
		}
		return toolResult(map[string]interface{}{"success": ok2, "data": data, "error": errMsg, "snapshot": snap}), nil
	}
}

func handleType(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		idxF, ok := args["elementIndex"].(float64)
		if !ok {
			return mcp.NewToolResultError("elementIndex is required"), nil
		}
		text, ok := args["text"].(string)
		if !ok || text == "" {
			return mcp.NewToolResultError("text is required"), nil
		}
		params := map[string]interface{}{"elementIndex": int(idxF), "text": text}
		if clear, ok := args["clear"].(bool); ok {
			params["clear"] = clear
		}
		ok2, data, errMsg, snap, err := h.BrowserInteract("type", params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("type failed: %v", err)), nil
		}
		if submit, _ := args["submit"].(bool); submit {
			okKey, _, keyErrMsg, keyErr := h.ExecuteScript("press_key.js", map[string]interface{}{"key": "Enter"})
			if keyErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("type succeeded but submit (Enter) failed: %v", keyErr)), nil
			}
			if !okKey {
				return mcp.NewToolResultError(fmt.Sprintf("type succeeded but submit (Enter) failed: %s", keyErrMsg)), nil
			}
		}
		return toolResult(map[string]interface{}{"success": ok2, "data": data, "error": errMsg, "snapshot": snap}), nil
	}
}

func handleHover(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		idxF, ok := args["elementIndex"].(float64)
		if !ok {
			return mcp.NewToolResultError("elementIndex is required"), nil
		}
		params := map[string]interface{}{"elementIndex": int(idxF)}
		ok2, data, errMsg, snap, err := h.BrowserInteract("hover", params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("hover failed: %v", err)), nil
		}
		return toolResult(map[string]interface{}{"success": ok2, "data": data, "error": errMsg, "snapshot": snap}), nil
	}
}

func handlePressKey(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		key, err := req.RequireString("key")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ok, data, errMsg, snap, execErr := h.ExecuteScriptAndObserve("press_key.js", map[string]interface{}{"key": key})
		if execErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("press_key failed: %v", execErr)), nil
		}
		return toolResult(map[string]interface{}{"success": ok, "data": data, "error": errMsg, "snapshot": snap}), nil
	}
}

func handleSelectOption(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		idxF, ok := args["elementIndex"].(float64)
		if !ok {
			return mcp.NewToolResultError("elementIndex is required"), nil
		}
		values, ok := args["values"].([]interface{})
		if !ok || len(values) == 0 {
			return mcp.NewToolResultError("values must be a non-empty array"), nil
		}
		ok2, data, errMsg, snap, err := h.ExecuteScriptAndObserve("select_option.js", map[string]interface{}{
			"elementIndex": int(idxF),
			"values":       values,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("select_option failed: %v", err)), nil
		}
		return toolResult(map[string]interface{}{"success": ok2, "data": data, "error": errMsg, "snapshot": snap}), nil
	}
}

func handleScroll(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		params := map[string]interface{}{}
		if idxF, ok := args["elementIndex"].(float64); ok {
			params["elementIndex"] = int(idxF)
		}
		if dx, ok := args["deltaX"].(float64); ok {
			params["deltaX"] = dx
		}
		if dy, ok := args["deltaY"].(float64); ok {
			params["deltaY"] = dy
		}
		ok, data, errMsg, snap, err := h.ExecuteScriptAndObserve("scroll.js", params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("scroll failed: %v", err)), nil
		}
		return toolResult(map[string]interface{}{"success": ok, "data": data, "error": errMsg, "snapshot": snap}), nil
	}
}

func handleDrag(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		startF, ok := args["startElementIndex"].(float64)
		if !ok {
			return mcp.NewToolResultError("startElementIndex is required"), nil
		}
		endF, ok := args["endElementIndex"].(float64)
		if !ok {
			return mcp.NewToolResultError("endElementIndex is required"), nil
		}
		ok2, data, errMsg, snap, err := h.ExecuteScriptAndObserve("drag.js", map[string]interface{}{
			"startElementIndex": int(startF),
			"endElementIndex":   int(endF),
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("drag failed: %v", err)), nil
		}
		return toolResult(map[string]interface{}{"success": ok2, "data": data, "error": errMsg, "snapshot": snap}), nil
	}
}

func handleFillForm(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		fields, ok := args["fields"].([]interface{})
		if !ok || len(fields) == 0 {
			return mcp.NewToolResultError("fields must be a non-empty array"), nil
		}
		ok2, data, errMsg, snap, err := h.ExecuteScriptAndObserve("fill_form.js", map[string]interface{}{"fields": fields})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("fill_form failed: %v", err)), nil
		}
		return toolResult(map[string]interface{}{"success": ok2, "data": data, "error": errMsg, "snapshot": snap}), nil
	}
}

func handleWaitFor(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		params := map[string]interface{}{}
		if t, ok := args["time"].(float64); ok {
			params["time"] = t
		}
		if text, ok := args["text"].(string); ok {
			params["text"] = text
		}
		if tg, ok := args["textGone"].(string); ok {
			params["textGone"] = tg
		}
		if to, ok := args["timeout"].(float64); ok {
			params["timeout"] = to
		}
		ok, data, errMsg, snap, err := h.ExecuteScriptAndObserve("wait_for.js", params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("wait_for failed: %v", err)), nil
		}
		return toolResult(map[string]interface{}{"success": ok, "data": data, "error": errMsg, "snapshot": snap}), nil
	}
}

func handleEvaluate(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		code, err := req.RequireString("code")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		params := map[string]interface{}{}
		if args := req.GetArguments(); args != nil {
			if p, ok := args["params"].(map[string]interface{}); ok {
				params = p
			}
		}
		ok, rawData, errMsg, snap, execErr := h.ExecuteRaw(code, params)
		if execErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("evaluate failed: %v", execErr)), nil
		}
		return toolResult(map[string]interface{}{"success": ok, "data": rawData, "error": errMsg, "snapshot": snap}), nil
	}
}

func handleTakeScreenshot(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dataURL, err := h.BrowserScreenshot()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("screenshot failed: %v", err)), nil
		}
		return mcp.NewToolResultText(dataURL), nil
	}
}

func handleTabs(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action, err := req.RequireString("action")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		args := req.GetArguments()
		var idx *int
		if args != nil {
			if idxF, ok := args["index"].(float64); ok {
				i := int(idxF)
				idx = &i
			}
		}
		urlStr := ""
		if args != nil {
			if u, ok := args["url"].(string); ok {
				urlStr = u
			}
		}
		// Validate URL scheme for create action.
		if action == "create" && urlStr != "" {
			parsed, parseErr := url.Parse(urlStr)
			if parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return mcp.NewToolResultError(fmt.Sprintf("invalid URL %q: only http and https are supported", urlStr)), nil
			}
		}
		ok, data, errMsg, execErr := h.BrowserTabs(action, idx, urlStr)
		if execErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("tabs failed: %v", execErr)), nil
		}
		return toolResult(map[string]interface{}{"success": ok, "data": data, "error": errMsg}), nil
	}
}

func handleExecuteScript(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		script, err := req.RequireString("script")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		params := map[string]interface{}{}
		if args := req.GetArguments(); args != nil {
			if p, ok := args["params"].(map[string]interface{}); ok {
				params = p
			}
		}
		ok, dataStr, errMsg, snap, execErr := h.ExecuteScriptAndObserve(script, params)
		if execErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("execution error: %v", execErr)), nil
		}
		return toolResult(map[string]interface{}{"success": ok, "data": dataStr, "error": errMsg, "snapshot": snap}), nil
	}
}

func handleListScripts(h *api.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		scripts, err := h.ListScripts()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list scripts failed: %v", err)), nil
		}
		return toolResult(map[string]interface{}{"scripts": scripts}), nil
	}
}
