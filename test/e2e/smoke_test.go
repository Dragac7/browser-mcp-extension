package e2e_test

import (
	"testing"
)

func TestSmokeConnection(t *testing.T) {
	snap := navigateTo(t, "basic.html")

	title, ok := snap["title"].(string)
	if !ok {
		t.Fatal("snapshot missing title")
	}
	if title != "E2E Basic Page" {
		t.Errorf("expected title %q, got %q", "E2E Basic Page", title)
	}

	mustContainText(t, snap, "Hello World")
}
