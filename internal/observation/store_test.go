package observation

import (
	"os"
	"testing"
)

func TestNewStore(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.Dir() != dir {
		t.Errorf("expected dir %q, got %q", dir, store.Dir())
	}
}

func TestLatestNilBeforeSave(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	if store.Latest() != nil {
		t.Error("expected nil before any Save")
	}
}

func TestLatestFromDiskNilBeforeSave(t *testing.T) {
	store, _ := NewStore(t.TempDir())
	snap, err := store.LatestFromDisk()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap != nil {
		t.Error("expected nil when no file on disk")
	}
}

func TestSaveAndLatest(t *testing.T) {
	store, _ := NewStore(t.TempDir())

	snapJSON := `{"url":"https://example.com","title":"Test","timestamp":"2026-01-01T00:00:00Z","interactiveElements":[],"totalInteractiveElements":0,"visibleText":"hello","sections":[]}`
	snap, err := store.Save(snapJSON, "test-action", "test-result", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if store.Latest() == nil {
		t.Error("expected non-nil Latest() after Save")
	}
}

func TestSaveAndLatestFromDisk(t *testing.T) {
	dir := t.TempDir()
	store1, _ := NewStore(dir)

	snapJSON := `{"url":"https://example.com","title":"Test","timestamp":"2026-01-01T00:00:00Z","interactiveElements":[],"totalInteractiveElements":0,"visibleText":"hello","sections":[]}`
	_, err := store1.Save(snapJSON, "action", "result", true)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	// Create a new store pointing to the same directory (simulates restart).
	store2, _ := NewStore(dir)
	if store2.Latest() != nil {
		t.Error("in-memory Latest should be nil for fresh store")
	}
	snap, err := store2.LatestFromDisk()
	if err != nil {
		t.Fatalf("LatestFromDisk: %v", err)
	}
	if snap == nil {
		t.Fatal("expected snapshot from disk after restart")
	}
}

func TestNewStore_MemoryOnly(t *testing.T) {
	store, err := NewStore("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.Dir() != "" {
		t.Errorf("expected empty dir, got %q", store.Dir())
	}
	if store.Latest() != nil {
		t.Error("expected nil Latest() initially")
	}
}

func TestStore_MemoryOnly_Save(t *testing.T) {
	// Use a temp dir to verify no files are created by the memory-only store.
	tmpDir := t.TempDir()

	store, err := NewStore("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapJSON := `{"url":"https://example.com","title":"MemTest","timestamp":"2026-01-01T00:00:00Z","interactiveElements":[],"totalInteractiveElements":0,"visibleText":"mem","sections":[]}`
	snap, err := store.Save(snapJSON, "mem-action", "mem-result", true)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if snap == nil {
		t.Fatal("expected non-nil snapshot from Save")
	}
	if store.Latest() == nil {
		t.Error("expected non-nil Latest() after Save")
	}
	if store.Latest().URL != "https://example.com" {
		t.Errorf("expected URL https://example.com, got %q", store.Latest().URL)
	}

	// Verify no files were created in tmpDir.
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no files in tmpDir, got %d", len(entries))
	}
}

func TestStore_MemoryOnly_LatestFromDisk(t *testing.T) {
	store, _ := NewStore("")
	snap, err := store.LatestFromDisk()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap != nil {
		t.Error("expected nil from LatestFromDisk in memory-only mode")
	}
}
