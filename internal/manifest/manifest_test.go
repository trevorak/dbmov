package manifest

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, FileName)
	m := &Manifest{
		ToolVersion:   "vtest",
		CreatedAt:     time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
		ServerVersion: "8.0.99",
		Entries: []Entry{
			{Database: "x", File: "x.sql", ExitCode: 0},
		},
	}
	if err := Write(p, m); err != nil {
		t.Fatal(err)
	}
	got, err := Read(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.ToolVersion != m.ToolVersion || got.ServerVersion != m.ServerVersion {
		t.Fatalf("%+v", got)
	}
	if len(got.Entries) != 1 || got.Entries[0].Database != "x" {
		t.Fatalf("%+v", got)
	}
}

func TestReadOrNil(t *testing.T) {
	dir := t.TempDir()
	m, err := ReadOrNil(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		t.Fatal("expected nil")
	}
	_ = os.WriteFile(filepath.Join(dir, FileName), []byte("{"), 0o644)
	if _, err := ReadOrNil(dir); err == nil {
		t.Fatal("expected error on bad json")
	}
}
