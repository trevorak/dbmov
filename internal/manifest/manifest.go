package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Manifest is written alongside per-database dump files for audit and restore.
type Manifest struct {
	ToolVersion   string    `json:"tool_version"`
	CreatedAt     time.Time `json:"created_at"`
	ServerVersion string    `json:"server_version"`
	Entries       []Entry   `json:"entries"`
}

// Entry describes one exported database.
type Entry struct {
	Database string `json:"database"`
	File     string `json:"file"` // relative path under backup root
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}

// FileName is the manifest filename stored in a backup directory.
const FileName = "manifest.json"

// Write encodes m to path (usually backupDir/manifest.json).
func Write(path string, m *Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Read loads a manifest from path.
func Read(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ReadOrNil tries to read manifest.json under dir; returns nil, nil if missing.
func ReadOrNil(dir string) (*Manifest, error) {
	p := filepath.Join(dir, FileName)
	if _, err := os.Stat(p); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return Read(p)
}
