package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/playwright-community/playwright-go"
)

var (
	mcpClient   *client.Client
	pageBaseURL string
	pw          *playwright.Playwright
	browserCtx  playwright.BrowserContext
)

func TestMain(m *testing.M) {
	// 1. Resolve paths — test/e2e → repo root
	testDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("cannot get working directory: %v", err)
	}
	repoRoot, err := filepath.Abs(filepath.Join(testDir, "..", ".."))
	if err != nil {
		log.Fatalf("cannot resolve repo root: %v", err)
	}
	extensionSrcDir := filepath.Join(repoRoot, "extension")
	scriptsPath := filepath.Join(repoRoot, "resources", "js_scripts")

	// 2. Run make sync-scripts with 60s timeout
	syncCtx, syncCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer syncCancel()
	syncCmd := exec.CommandContext(syncCtx, "make", "sync-scripts")
	syncCmd.Dir = repoRoot
	syncCmd.Stdout = os.Stdout
	syncCmd.Stderr = os.Stderr
	if err := syncCmd.Run(); err != nil {
		log.Fatalf("make sync-scripts failed: %v", err)
	}

	// 3. Build binary with 120s timeout
	tmpDir, err := os.MkdirTemp("", "browser-e2e-*")
	if err != nil {
		log.Fatalf("cannot create temp dir: %v", err)
	}
	binaryPath := filepath.Join(tmpDir, "browser-cmd")
	buildCtx, buildCancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer buildCancel()
	buildCmd := exec.CommandContext(buildCtx, "go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		log.Fatalf("go build failed: %v", err)
	}

	// 4. Copy extension to temp dir using os.CopyFS
	tempExtDir := filepath.Join(tmpDir, "extension")
	if err := os.MkdirAll(tempExtDir, 0o755); err != nil {
		log.Fatalf("cannot create temp extension dir: %v", err)
	}
	if err := os.CopyFS(tempExtDir, os.DirFS(extensionSrcDir)); err != nil {
		log.Fatalf("cannot copy extension to temp dir: %v", err)
	}

	// 5. Write config.json with WS port 19001
	configJSON := []byte(`{"port": 19001}`)
	if err := os.WriteFile(filepath.Join(tempExtDir, "config.json"), configJSON, 0o644); err != nil {
		log.Fatalf("cannot write config.json: %v", err)
	}

	// 6. Start httptest server serving testdata/
	fileServer := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	pageBaseURL = fileServer.URL + "/"

	// 7. Install playwright browsers (chromium only)
	if err := playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}}); err != nil {
		log.Fatalf("playwright install failed: %v", err)
	}

	// 8. Start playwright
	pw, err = playwright.Run()
	if err != nil {
		log.Fatalf("playwright run failed: %v", err)
	}

	// 9. Launch browser with extension
	userDataDir := filepath.Join(tmpDir, "userdata")
	browserCtx, err = pw.Chromium.LaunchPersistentContext(userDataDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(false),
		Args: []string{
			fmt.Sprintf("--disable-extensions-except=%s", tempExtDir),
			fmt.Sprintf("--load-extension=%s", tempExtDir),
			"--no-first-run",
			"--disable-gpu",
		},
	})
	if err != nil {
		log.Fatalf("browser launch failed: %v", err)
	}

	// 10. Start MCP client
	observationsDir := filepath.Join(tmpDir, "observations")
	if err := os.MkdirAll(observationsDir, 0o755); err != nil {
		log.Fatalf("cannot create observations dir: %v", err)
	}
	mcpClient, err = client.NewStdioMCPClient(
		binaryPath,
		[]string{
			"WS_PORT=19001",
			fmt.Sprintf("JS_SCRIPTS_PATH=%s", scriptsPath),
			fmt.Sprintf("OBSERVATIONS_DIR=%s", observationsDir),
		},
		"--mcp",
	)
	if err != nil {
		log.Fatalf("MCP client start failed: %v", err)
	}

	// 11. Initialize MCP
	ctx := context.Background()
	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "e2e-test",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		log.Fatalf("MCP initialize failed: %v", err)
	}

	// 12. Wait for WS connection: poll browser_tabs (action=list) every 1s, max 30s
	deadline := time.Now().Add(30 * time.Second)
	connected := false
	for time.Now().Before(deadline) {
		result, callErr := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "browser_tabs",
				Arguments: map[string]interface{}{"action": "list"},
			},
		})
		if callErr == nil && result != nil && !result.IsError {
			connected = true
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !connected {
		log.Fatalf("extension did not connect to WebSocket within 30s")
	}

	// 13. Run tests
	code := m.Run()

	// 14. Teardown
	if mcpClient != nil {
		mcpClient.Close()
	}
	if browserCtx != nil {
		browserCtx.Close()
	}
	if pw != nil {
		pw.Stop()
	}
	fileServer.Close()
	os.RemoveAll(tmpDir)

	os.Exit(code)
}

// callTool calls an MCP tool and returns the result. Fails the test on transport error.
func callTool(t *testing.T, toolName string, args map[string]interface{}) *mcp.CallToolResult {
	t.Helper()
	result, err := mcpClient.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
	if err != nil {
		t.Fatalf("callTool(%s) transport error: %v", toolName, err)
	}
	return result
}

// callToolExpectError calls an MCP tool, asserts IsError is true, returns the error text.
func callToolExpectError(t *testing.T, toolName string, args map[string]interface{}) string {
	t.Helper()
	result := callTool(t, toolName, args)
	if !result.IsError {
		t.Fatalf("callToolExpectError(%s): expected IsError=true, got false. Content: %v", toolName, result.Content)
	}
	if len(result.Content) == 0 {
		t.Fatalf("callToolExpectError(%s): no content in error result", toolName)
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("callToolExpectError(%s): content[0] is not TextContent: %T", toolName, result.Content[0])
	}
	return text.Text
}

// parseToolResult extracts text content from a CallToolResult and unmarshals it as JSON.
func parseToolResult(t *testing.T, result *mcp.CallToolResult) map[string]interface{} {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("parseToolResult: empty content")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("parseToolResult: content[0] is not TextContent: %T", result.Content[0])
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(text.Text), &m); err != nil {
		t.Fatalf("parseToolResult: JSON unmarshal failed: %v\nRaw: %s", err, text.Text)
	}
	return m
}

// pageURL returns the full URL for a static test page.
func pageURL(pageName string) string {
	return pageBaseURL + "pages/" + pageName
}

// navigateTo navigates to a test page via browser_navigate and returns the parsed result.
func navigateTo(t *testing.T, pageName string) map[string]interface{} {
	t.Helper()
	result := callTool(t, "browser_navigate", map[string]interface{}{
		"url": pageURL(pageName),
	})
	if result.IsError {
		t.Fatalf("navigateTo(%s) failed: %v", pageName, result.Content)
	}
	return parseToolResult(t, result)
}

// snapshotPage calls browser_snapshot and returns the parsed result.
func snapshotPage(t *testing.T) map[string]interface{} {
	t.Helper()
	result := callTool(t, "browser_snapshot", nil)
	if result.IsError {
		t.Fatalf("snapshotPage failed: %v", result.Content)
	}
	return parseToolResult(t, result)
}

// mustContainText asserts that snap["visibleText"] contains the given substring.
func mustContainText(t *testing.T, snap map[string]interface{}, text string) {
	t.Helper()
	visible, _ := snap["visibleText"].(string)
	if !strings.Contains(visible, text) {
		t.Errorf("expected visibleText to contain %q, got %q", text, visible)
	}
}

// getToolResultText extracts the raw text string from a CallToolResult without JSON parsing.
// Use this for tools that return non-JSON text (e.g., browser_take_screenshot returns a data URL).
func getToolResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("getToolResultText: empty content")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("getToolResultText: content[0] is not TextContent: %T", result.Content[0])
	}
	return text.Text
}

// snapshotHasElement scans interactiveElements for an element whose JSON representation
// contains the given text substring. Returns the array index and true if found.
func snapshotHasElement(snap map[string]interface{}, textSubstring string) (int, bool) {
	elements, ok := snap["interactiveElements"].([]interface{})
	if !ok {
		return -1, false
	}
	for i, el := range elements {
		data, err := json.Marshal(el)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), textSubstring) {
			return i, true
		}
	}
	return -1, false
}
