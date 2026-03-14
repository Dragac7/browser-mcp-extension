package observation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Snapshot represents a structured page observation captured after an action.
type Snapshot struct {
	URL                      string            `json:"url"`
	Title                    string            `json:"title"`
	Timestamp                string            `json:"timestamp"`
	InteractiveElements      []json.RawMessage `json:"interactiveElements"`
	TotalInteractiveElements int               `json:"totalInteractiveElements"`
	VisibleText              string            `json:"visibleText"`
	Sections                 []json.RawMessage `json:"sections"`
	Action                   string            `json:"action,omitempty"`
	ActionResult             string            `json:"actionResult,omitempty"`
	ActionSuccess            bool              `json:"actionSuccess,omitempty"`
}

// Store manages saving and retrieving page snapshots on disk.
type Store struct {
	dir    string
	mu     sync.RWMutex
	latest *Snapshot
}

// NewStore creates a Store that persists observations in the given directory.
// When dir is empty, the store operates in memory-only mode (no disk writes).
// The directory is created if it does not exist.
func NewStore(dir string) (*Store, error) {
	if dir == "" {
		return &Store{}, nil
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("observation: bad path %q: %w", dir, err)
	}
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, fmt.Errorf("observation: cannot create dir %q: %w", absDir, err)
	}
	return &Store{dir: absDir}, nil
}

// Save persists a snapshot to latest.json and a timestamped history file.
// The raw JSON string from observe.js is parsed, enriched with action metadata,
// and written to disk.
func (s *Store) Save(rawJSON string, action string, actionResult string, actionSuccess bool) (*Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var snap Snapshot
	if err := json.Unmarshal([]byte(rawJSON), &snap); err != nil {
		return nil, fmt.Errorf("observation: invalid snapshot JSON: %w", err)
	}

	snap.Action = action
	snap.ActionResult = actionResult
	snap.ActionSuccess = actionSuccess

	if s.dir != "" {
		data, err := json.MarshalIndent(snap, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("observation: marshal error: %w", err)
		}
		latestPath := filepath.Join(s.dir, "latest.json")
		if err := os.WriteFile(latestPath, data, 0o644); err != nil {
			return nil, fmt.Errorf("observation: write latest.json: %w", err)
		}
		ts := time.Now().Format("2006-01-02T15-04-05")
		historyPath := filepath.Join(s.dir, fmt.Sprintf("%s.json", ts))
		if err := os.WriteFile(historyPath, data, 0o644); err != nil {
			return nil, fmt.Errorf("observation: write history: %w", err)
		}
	}

	s.latest = &snap
	return &snap, nil
}

// Latest returns the most recently saved snapshot (from memory).
// Returns nil if no snapshot has been saved yet in this session.
func (s *Store) Latest() *Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latest
}

// LatestFromDisk reads and returns the latest.json from disk.
// Returns nil, nil when the store is in memory-only mode or no snapshot exists.
func (s *Store) LatestFromDisk() (*Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.dir == "" {
		return nil, nil
	}

	data, err := os.ReadFile(filepath.Join(s.dir, "latest.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("observation: read latest.json: %w", err)
	}

	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("observation: parse latest.json: %w", err)
	}

	return &snap, nil
}

// Dir returns the absolute path to the observations directory.
func (s *Store) Dir() string {
	return s.dir
}
