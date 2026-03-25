package mcpserver

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/paoloandrisani/browser-mcp-extension/internal/api"
	"github.com/paoloandrisani/browser-mcp-extension/internal/observation"
)

func buildUploadTestHandler(t *testing.T, executeFile api.ExecuteFileFn) *api.Handler {
	t.Helper()
	dir := t.TempDir()
	store, _ := observation.NewStore(dir)
	scriptsDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("cannot create scripts dir: %v", err)
	}
	execute := func(code string) (bool, interface{}, string, error) { return true, "ok", "", nil }
	screenshot := func() (string, error) { return "data:image/png;base64,abc", nil }
	tabs := func(action string, index *int, url string) (bool, interface{}, string, error) {
		return true, nil, "", nil
	}
	return api.NewHandler(store, execute, executeFile, screenshot, tabs, scriptsDir, "")
}

func makeUploadRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "browser_upload",
			Arguments: args,
		},
	}
}

func TestHandleUpload_MissingElementIndex(t *testing.T) {
	h := buildUploadTestHandler(t, func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		return true, "ok", "", nil
	})
	handler := handleUpload(h)
	result, err := handler(context.Background(), makeUploadRequest(map[string]interface{}{
		"files": []interface{}{map[string]interface{}{"name": "test.txt", "content": "aGVsbG8="}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(text.Text, "elementIndex is required") {
		t.Errorf("expected error to contain 'elementIndex is required', got %q", text.Text)
	}
}

func TestHandleUpload_MissingFiles(t *testing.T) {
	h := buildUploadTestHandler(t, func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		return true, "ok", "", nil
	})
	handler := handleUpload(h)
	result, err := handler(context.Background(), makeUploadRequest(map[string]interface{}{
		"elementIndex": float64(0),
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(text.Text, "files must be a non-empty array") {
		t.Errorf("expected error to contain 'files must be a non-empty array', got %q", text.Text)
	}
}

func TestHandleUpload_EmptyFiles(t *testing.T) {
	h := buildUploadTestHandler(t, func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		return true, "ok", "", nil
	})
	handler := handleUpload(h)
	result, err := handler(context.Background(), makeUploadRequest(map[string]interface{}{
		"elementIndex": float64(0),
		"files":        []interface{}{},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(text.Text, "files must be a non-empty array") {
		t.Errorf("expected error to contain 'files must be a non-empty array', got %q", text.Text)
	}
}

func TestHandleUpload_Success(t *testing.T) {
	var uploadParams map[string]interface{}
	dir := t.TempDir()
	store, _ := observation.NewStore(dir)
	scriptsDir := filepath.Join(dir, "scripts")
	os.MkdirAll(scriptsDir, 0o755)
	os.WriteFile(filepath.Join(scriptsDir, "upload.js"), []byte("return 'ok';"), 0o644)
	os.WriteFile(filepath.Join(scriptsDir, "observe.js"), []byte("return {};"), 0o644)

	executeFile := func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		if strings.Contains(scriptFile, "upload.js") {
			uploadParams = params
		}
		return true, "Uploaded 1 file(s)", "", nil
	}
	execute := func(code string) (bool, interface{}, string, error) { return true, "ok", "", nil }
	screenshot := func() (string, error) { return "", nil }
	tabs := func(action string, index *int, url string) (bool, interface{}, string, error) {
		return true, nil, "", nil
	}
	h := api.NewHandler(store, execute, executeFile, screenshot, tabs, scriptsDir, "")
	fn := handleUpload(h)

	result, err := fn(context.Background(), makeUploadRequest(map[string]interface{}{
		"elementIndex": float64(2),
		"files":        []interface{}{map[string]interface{}{"name": "test.txt", "content": "aGVsbG8="}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected IsError=false, got error: %v", result.Content)
	}

	if uploadParams == nil {
		t.Fatal("executeFile was not called for upload.js")
	}
	if idx, ok := uploadParams["elementIndex"].(int); !ok || idx != 2 {
		t.Errorf("expected elementIndex=2, got %v", uploadParams["elementIndex"])
	}
}

func TestHandleUpload_ExecuteError(t *testing.T) {
	dir := t.TempDir()
	store, _ := observation.NewStore(dir)
	scriptsDir := filepath.Join(dir, "scripts")
	os.MkdirAll(scriptsDir, 0o755)
	os.WriteFile(filepath.Join(scriptsDir, "upload.js"), []byte("return 'ok';"), 0o644)

	executeFile := func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		return false, nil, "", fmt.Errorf("connection lost")
	}
	execute := func(code string) (bool, interface{}, string, error) { return true, "ok", "", nil }
	screenshot := func() (string, error) { return "", nil }
	tabs := func(action string, index *int, url string) (bool, interface{}, string, error) {
		return true, nil, "", nil
	}
	h := api.NewHandler(store, execute, executeFile, screenshot, tabs, scriptsDir, "")
	fn := handleUpload(h)

	result, err := fn(context.Background(), makeUploadRequest(map[string]interface{}{
		"elementIndex": float64(0),
		"files":        []interface{}{map[string]interface{}{"name": "test.txt", "content": "aGVsbG8="}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(text.Text, "upload failed") {
		t.Errorf("expected error to contain 'upload failed', got %q", text.Text)
	}
}

func TestHandleUpload_FilePath_Success(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	fileContent := []byte("hello from file")
	os.WriteFile(tmpFile, fileContent, 0o644)

	var capturedParams map[string]interface{}
	dir := t.TempDir()
	store, _ := observation.NewStore(dir)
	scriptsDir := filepath.Join(dir, "scripts")
	os.MkdirAll(scriptsDir, 0o755)
	os.WriteFile(filepath.Join(scriptsDir, "upload.js"), []byte("return 'ok';"), 0o644)
	os.WriteFile(filepath.Join(scriptsDir, "observe.js"), []byte("return {};"), 0o644)

	executeFile := func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		if strings.Contains(scriptFile, "upload.js") {
			capturedParams = params
		}
		return true, "Uploaded 1 file(s)", "", nil
	}
	execute := func(code string) (bool, interface{}, string, error) { return true, "ok", "", nil }
	screenshot := func() (string, error) { return "", nil }
	tabs := func(action string, index *int, url string) (bool, interface{}, string, error) {
		return true, nil, "", nil
	}
	h := api.NewHandler(store, execute, executeFile, screenshot, tabs, scriptsDir, "")
	fn := handleUpload(h)

	result, err := fn(context.Background(), makeUploadRequest(map[string]interface{}{
		"elementIndex": float64(0),
		"files": []interface{}{map[string]interface{}{
			"name":     "test.txt",
			"filePath": tmpFile,
		}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected IsError=false, got error: %v", result.Content)
	}
	if capturedParams == nil {
		t.Fatal("executeFile was not called for upload.js")
	}
	files, ok := capturedParams["files"].([]interface{})
	if !ok || len(files) == 0 {
		t.Fatal("expected files in captured params")
	}
	entry, _ := files[0].(map[string]interface{})
	content, _ := entry["content"].(string)
	expected := base64.StdEncoding.EncodeToString(fileContent)
	if content != expected {
		t.Errorf("expected base64 content %q, got %q", expected, content)
	}
	if _, hasFilePath := entry["filePath"]; hasFilePath {
		t.Error("filePath should have been removed from entry")
	}
}

func TestHandleUpload_FilePath_AutoName(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "my_document.pdf")
	os.WriteFile(tmpFile, []byte("pdf content"), 0o644)

	var capturedParams map[string]interface{}
	dir := t.TempDir()
	store, _ := observation.NewStore(dir)
	scriptsDir := filepath.Join(dir, "scripts")
	os.MkdirAll(scriptsDir, 0o755)
	os.WriteFile(filepath.Join(scriptsDir, "upload.js"), []byte("return 'ok';"), 0o644)
	os.WriteFile(filepath.Join(scriptsDir, "observe.js"), []byte("return {};"), 0o644)

	executeFile := func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		if strings.Contains(scriptFile, "upload.js") {
			capturedParams = params
		}
		return true, "Uploaded 1 file(s)", "", nil
	}
	execute := func(code string) (bool, interface{}, string, error) { return true, "ok", "", nil }
	screenshot := func() (string, error) { return "", nil }
	tabs := func(action string, index *int, url string) (bool, interface{}, string, error) {
		return true, nil, "", nil
	}
	h := api.NewHandler(store, execute, executeFile, screenshot, tabs, scriptsDir, "")
	fn := handleUpload(h)

	result, err := fn(context.Background(), makeUploadRequest(map[string]interface{}{
		"elementIndex": float64(0),
		"files": []interface{}{map[string]interface{}{
			"filePath": tmpFile,
		}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected IsError=false, got error: %v", result.Content)
	}
	files, _ := capturedParams["files"].([]interface{})
	entry, _ := files[0].(map[string]interface{})
	name, _ := entry["name"].(string)
	if name != "my_document.pdf" {
		t.Errorf("expected auto-derived name 'my_document.pdf', got %q", name)
	}
}

func TestHandleUpload_FilePath_MimeInference(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "resume.pdf")
	os.WriteFile(tmpFile, []byte("pdf"), 0o644)

	var capturedParams map[string]interface{}
	dir := t.TempDir()
	store, _ := observation.NewStore(dir)
	scriptsDir := filepath.Join(dir, "scripts")
	os.MkdirAll(scriptsDir, 0o755)
	os.WriteFile(filepath.Join(scriptsDir, "upload.js"), []byte("return 'ok';"), 0o644)
	os.WriteFile(filepath.Join(scriptsDir, "observe.js"), []byte("return {};"), 0o644)

	executeFile := func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		if strings.Contains(scriptFile, "upload.js") {
			capturedParams = params
		}
		return true, "ok", "", nil
	}
	execute := func(code string) (bool, interface{}, string, error) { return true, "ok", "", nil }
	screenshot := func() (string, error) { return "", nil }
	tabs := func(action string, index *int, url string) (bool, interface{}, string, error) {
		return true, nil, "", nil
	}
	h := api.NewHandler(store, execute, executeFile, screenshot, tabs, scriptsDir, "")
	fn := handleUpload(h)

	// Without explicit mimeType → should infer application/pdf
	result, err := fn(context.Background(), makeUploadRequest(map[string]interface{}{
		"elementIndex": float64(0),
		"files": []interface{}{map[string]interface{}{
			"name":     "resume.pdf",
			"filePath": tmpFile,
		}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}
	files, _ := capturedParams["files"].([]interface{})
	entry, _ := files[0].(map[string]interface{})
	mime, _ := entry["mimeType"].(string)
	if mime != "application/pdf" {
		t.Errorf("expected inferred mimeType 'application/pdf', got %q", mime)
	}

	// With explicit mimeType → should NOT be overridden
	tmpFile2 := filepath.Join(t.TempDir(), "data.pdf")
	os.WriteFile(tmpFile2, []byte("pdf"), 0o644)
	result, err = fn(context.Background(), makeUploadRequest(map[string]interface{}{
		"elementIndex": float64(0),
		"files": []interface{}{map[string]interface{}{
			"name":     "data.pdf",
			"filePath": tmpFile2,
			"mimeType": "custom/type",
		}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	files, _ = capturedParams["files"].([]interface{})
	entry, _ = files[0].(map[string]interface{})
	mime, _ = entry["mimeType"].(string)
	if mime != "custom/type" {
		t.Errorf("expected explicit mimeType 'custom/type' to be preserved, got %q", mime)
	}
}

func TestHandleUpload_FilePath_RelativePath(t *testing.T) {
	h := buildUploadTestHandler(t, func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		return true, "ok", "", nil
	})
	fn := handleUpload(h)
	result, err := fn(context.Background(), makeUploadRequest(map[string]interface{}{
		"elementIndex": float64(0),
		"files": []interface{}{map[string]interface{}{
			"name":     "test.txt",
			"filePath": "relative/path.txt",
		}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	text, _ := result.Content[0].(mcp.TextContent)
	if !strings.Contains(text.Text, "must be absolute") {
		t.Errorf("expected error to contain 'must be absolute', got %q", text.Text)
	}
}

func TestHandleUpload_FilePath_FileNotFound(t *testing.T) {
	h := buildUploadTestHandler(t, func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		return true, "ok", "", nil
	})
	fn := handleUpload(h)
	result, err := fn(context.Background(), makeUploadRequest(map[string]interface{}{
		"elementIndex": float64(0),
		"files": []interface{}{map[string]interface{}{
			"name":     "test.txt",
			"filePath": "/nonexistent/path/file.txt",
		}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	text, _ := result.Content[0].(mcp.TextContent)
	if !strings.Contains(text.Text, "cannot read file") {
		t.Errorf("expected error to contain 'cannot read file', got %q", text.Text)
	}
}

func TestHandleUpload_FilePath_TooLarge(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "large.bin")
	f, _ := os.Create(tmpFile)
	f.Truncate(11 * 1024 * 1024) // 11 MB sparse file
	f.Close()

	h := buildUploadTestHandler(t, func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		return true, "ok", "", nil
	})
	fn := handleUpload(h)
	result, err := fn(context.Background(), makeUploadRequest(map[string]interface{}{
		"elementIndex": float64(0),
		"files": []interface{}{map[string]interface{}{
			"name":     "large.bin",
			"filePath": tmpFile,
		}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	text, _ := result.Content[0].(mcp.TextContent)
	if !strings.Contains(text.Text, "exceeds 10 MB") {
		t.Errorf("expected error to contain 'exceeds 10 MB', got %q", text.Text)
	}
}

func TestHandleUpload_FilePath_BothContentAndFilePath(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmpFile, []byte("hello"), 0o644)

	h := buildUploadTestHandler(t, func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		return true, "ok", "", nil
	})
	fn := handleUpload(h)
	result, err := fn(context.Background(), makeUploadRequest(map[string]interface{}{
		"elementIndex": float64(0),
		"files": []interface{}{map[string]interface{}{
			"name":     "test.txt",
			"content":  "aGVsbG8=",
			"filePath": tmpFile,
		}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	text, _ := result.Content[0].(mcp.TextContent)
	if !strings.Contains(text.Text, "not both") {
		t.Errorf("expected error to contain 'not both', got %q", text.Text)
	}
}

func TestHandleUpload_FilePath_NeitherContentNorFilePath(t *testing.T) {
	h := buildUploadTestHandler(t, func(scriptFile, code string, params map[string]interface{}) (bool, interface{}, string, error) {
		return true, "ok", "", nil
	})
	fn := handleUpload(h)
	result, err := fn(context.Background(), makeUploadRequest(map[string]interface{}{
		"elementIndex": float64(0),
		"files":        []interface{}{map[string]interface{}{"name": "test.txt"}},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	text, _ := result.Content[0].(mcp.TextContent)
	if !strings.Contains(text.Text, "content or filePath is required") {
		t.Errorf("expected error to contain 'content or filePath is required', got %q", text.Text)
	}
}
