package e2e_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpload_SingleFile(t *testing.T) {
	navigateTo(t, "upload.html")
	snap := snapshotPage(t)

	idx, found := snapshotHasElement(snap, "Single file")
	if !found {
		t.Fatal("could not find 'Single file' input")
	}

	content := base64.StdEncoding.EncodeToString([]byte("hello world"))
	result := callTool(t, "browser_upload", map[string]interface{}{
		"elementIndex": idx,
		"files": []interface{}{
			map[string]interface{}{
				"name":    "test.txt",
				"content": content,
			},
		},
	})
	parsed := parseToolResult(t, result)

	success, _ := parsed["success"].(bool)
	if !success {
		t.Fatalf("upload failed: data=%v error=%v", parsed["data"], parsed["error"])
	}

	data, _ := parsed["data"].(string)
	if !strings.Contains(data, "Uploaded 1 file(s)") {
		t.Errorf("expected 'Uploaded 1 file(s)' in data, got %q", data)
	}

	snap = snapshotPage(t)
	mustContainText(t, snap, "Selected: test.txt(11b)")
}

func TestUpload_MultipleFiles(t *testing.T) {
	navigateTo(t, "upload.html")
	snap := snapshotPage(t)

	idx, found := snapshotHasElement(snap, "Multiple files")
	if !found {
		t.Fatal("could not find 'Multiple files' input")
	}

	file1 := base64.StdEncoding.EncodeToString([]byte("file one"))
	file2 := base64.StdEncoding.EncodeToString([]byte("file two"))

	result := callTool(t, "browser_upload", map[string]interface{}{
		"elementIndex": idx,
		"files": []interface{}{
			map[string]interface{}{"name": "a.txt", "content": file1},
			map[string]interface{}{"name": "b.pdf", "content": file2},
		},
	})
	parsed := parseToolResult(t, result)

	success, _ := parsed["success"].(bool)
	if !success {
		t.Fatalf("upload failed: data=%v error=%v", parsed["data"], parsed["error"])
	}

	data, _ := parsed["data"].(string)
	if !strings.Contains(data, "Uploaded 2 file(s)") {
		t.Errorf("expected 'Uploaded 2 file(s)' in data, got %q", data)
	}

	snap = snapshotPage(t)
	mustContainText(t, snap, "Selected: a.txt(8b), b.pdf(8b)")
}

func TestUpload_InvalidIndex(t *testing.T) {
	navigateTo(t, "upload.html")
	snapshotPage(t)

	content := base64.StdEncoding.EncodeToString([]byte("test"))
	result := callTool(t, "browser_upload", map[string]interface{}{
		"elementIndex": 9999,
		"files": []interface{}{
			map[string]interface{}{"name": "test.txt", "content": content},
		},
	})
	parsed := parseToolResult(t, result)

	data, _ := parsed["data"].(string)
	if !strings.Contains(data, "out of range") {
		t.Errorf("expected data to contain 'out of range', got %q", data)
	}
}

func TestUpload_NonFileInput(t *testing.T) {
	// Navigate to basic.html which has non-file interactive elements (buttons, text inputs)
	navigateTo(t, "basic.html")
	snap := snapshotPage(t)

	// Find a button or text input — any non-file-input interactive element
	idx, found := snapshotHasElement(snap, "Change Text")
	if !found {
		t.Fatal("could not find a non-file-input element on basic.html")
	}

	content := base64.StdEncoding.EncodeToString([]byte("test"))
	result := callTool(t, "browser_upload", map[string]interface{}{
		"elementIndex": idx,
		"files": []interface{}{
			map[string]interface{}{"name": "test.txt", "content": content},
		},
	})
	parsed := parseToolResult(t, result)

	data, _ := parsed["data"].(string)
	if !strings.Contains(data, "not a file input") {
		t.Errorf("expected data to contain 'not a file input', got %q", data)
	}
}

func TestUpload_InvalidBase64(t *testing.T) {
	navigateTo(t, "upload.html")
	snap := snapshotPage(t)

	idx, found := snapshotHasElement(snap, "Single file")
	if !found {
		t.Fatal("could not find 'Single file' input")
	}

	result := callTool(t, "browser_upload", map[string]interface{}{
		"elementIndex": idx,
		"files": []interface{}{
			map[string]interface{}{"name": "test.txt", "content": "!!!not-valid-base64!!!"},
		},
	})
	parsed := parseToolResult(t, result)

	data, _ := parsed["data"].(string)
	if !strings.Contains(data, "not valid base64") {
		t.Errorf("expected data to contain 'not valid base64', got %q", data)
	}
}

func TestUpload_EmptyFilesArray(t *testing.T) {
	errText := callToolExpectError(t, "browser_upload", map[string]interface{}{
		"elementIndex": 0,
		"files":        []interface{}{},
	})
	if !strings.Contains(errText, "files must be a non-empty array") {
		t.Errorf("expected error to contain 'files must be a non-empty array', got %q", errText)
	}
}

func TestUpload_MultipleFilesOnSingleInput(t *testing.T) {
	navigateTo(t, "upload.html")
	snap := snapshotPage(t)

	idx, found := snapshotHasElement(snap, "Single file")
	if !found {
		t.Fatal("could not find 'Single file' input")
	}

	file1 := base64.StdEncoding.EncodeToString([]byte("one"))
	file2 := base64.StdEncoding.EncodeToString([]byte("two"))

	result := callTool(t, "browser_upload", map[string]interface{}{
		"elementIndex": idx,
		"files": []interface{}{
			map[string]interface{}{"name": "a.txt", "content": file1},
			map[string]interface{}{"name": "b.txt", "content": file2},
		},
	})
	parsed := parseToolResult(t, result)

	data, _ := parsed["data"].(string)
	if !strings.Contains(data, "multiple") {
		t.Errorf("expected data to contain 'multiple', got %q", data)
	}
}

func TestUpload_AcceptRestrictedInput(t *testing.T) {
	navigateTo(t, "upload.html")
	snap := snapshotPage(t)

	idx, found := snapshotHasElement(snap, "PDF or TXT only")
	if !found {
		t.Fatal("could not find 'PDF or TXT only' input")
	}

	content := base64.StdEncoding.EncodeToString([]byte("text content"))
	result := callTool(t, "browser_upload", map[string]interface{}{
		"elementIndex": idx,
		"files": []interface{}{
			map[string]interface{}{
				"name":     "readme.txt",
				"content":  content,
				"mimeType": "text/plain",
			},
		},
	})
	parsed := parseToolResult(t, result)

	success, _ := parsed["success"].(bool)
	if !success {
		t.Fatalf("upload failed: data=%v error=%v", parsed["data"], parsed["error"])
	}

	data, _ := parsed["data"].(string)
	if !strings.Contains(data, "readme.txt") {
		t.Errorf("expected data to contain 'readme.txt', got %q", data)
	}

	snap = snapshotPage(t)
	mustContainText(t, snap, "Selected: readme.txt(12b)")
}

func TestUpload_FilePath(t *testing.T) {
	navigateTo(t, "upload.html")
	snap := snapshotPage(t)

	idx, found := snapshotHasElement(snap, "Single file")
	if !found {
		t.Fatal("could not find 'Single file' input")
	}

	// Create a temp file with known content
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "upload_test.txt")
	fileContent := []byte("filePath upload works")
	if err := os.WriteFile(tmpFile, fileContent, 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	result := callTool(t, "browser_upload", map[string]interface{}{
		"elementIndex": idx,
		"files": []interface{}{
			map[string]interface{}{
				"name":     "upload_test.txt",
				"filePath": tmpFile,
			},
		},
	})
	parsed := parseToolResult(t, result)

	success, _ := parsed["success"].(bool)
	if !success {
		t.Fatalf("upload failed: data=%v error=%v", parsed["data"], parsed["error"])
	}

	data, _ := parsed["data"].(string)
	if !strings.Contains(data, "Uploaded 1 file(s)") {
		t.Errorf("expected 'Uploaded 1 file(s)' in data, got %q", data)
	}

	snap = snapshotPage(t)
	// "filePath upload works" is 21 bytes
	mustContainText(t, snap, "Selected: upload_test.txt(21b)")
}
