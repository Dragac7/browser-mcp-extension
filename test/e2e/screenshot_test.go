package e2e_test

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestScreenshot_ReturnsImage(t *testing.T) {
	navigateTo(t, "screenshot.html")

	result := callTool(t, "browser_take_screenshot", nil)
	if result.IsError {
		t.Fatalf("screenshot failed: %v", result.Content)
	}

	text := getToolResultText(t, result)
	if !strings.HasPrefix(text, "data:image/png;base64,") {
		t.Errorf("expected data URL prefix 'data:image/png;base64,', got prefix %q", text[:min(50, len(text))])
	}
	if len(text) <= 100 {
		t.Errorf("expected data URL length > 100, got %d", len(text))
	}
}

func TestScreenshot_ValidPNG(t *testing.T) {
	navigateTo(t, "screenshot.html")

	result := callTool(t, "browser_take_screenshot", nil)
	if result.IsError {
		t.Fatalf("screenshot failed: %v", result.Content)
	}

	text := getToolResultText(t, result)

	prefix := "data:image/png;base64,"
	if !strings.HasPrefix(text, prefix) {
		t.Fatalf("expected data URL prefix %q", prefix)
	}

	b64Data := text[len(prefix):]
	decoded, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}

	// PNG magic bytes: 0x89 0x50 0x4E 0x47
	if len(decoded) < 4 {
		t.Fatal("decoded data too short to be a valid PNG")
	}
	if decoded[0] != 0x89 || decoded[1] != 0x50 || decoded[2] != 0x4E || decoded[3] != 0x47 {
		t.Errorf("expected PNG magic bytes [0x89 0x50 0x4E 0x47], got [0x%02X 0x%02X 0x%02X 0x%02X]",
			decoded[0], decoded[1], decoded[2], decoded[3])
	}
}
