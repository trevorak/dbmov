package dump

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestDatabaseFromDumpFilename(t *testing.T) {
	tests := []struct {
		base string
		want string
	}{
		{"foo.sql", "foo"},
		{"foo.sql.gz", "foo"},
		{"my_schema.sql", "my_schema"},
		{"a.sql.gz", "a"},
		{"nope.txt", ""},
		{".sql", ""},
	}
	for _, tt := range tests {
		got := DatabaseFromDumpFilename(tt.base)
		if got != tt.want {
			t.Errorf("DatabaseFromDumpFilename(%q) = %q; want %q", tt.base, got, tt.want)
		}
	}
}

func TestOpenDumpReader(t *testing.T) {
	dir := t.TempDir()

	plain := filepath.Join(dir, "x.sql")
	if err := os.WriteFile(plain, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := OpenDumpReader(plain)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if closeErr := r.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hello" {
		t.Fatalf("plain read %q", b)
	}

	gzPath := filepath.Join(dir, "y.sql.gz")
	gzf, err := os.Create(gzPath)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(gzf)
	if _, err := gw.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzf.Close(); err != nil {
		t.Fatal(err)
	}

	r2, err := OpenDumpReader(gzPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Close()
	b2, err := io.ReadAll(r2)
	if err != nil {
		t.Fatal(err)
	}
	if string(b2) != "payload" {
		t.Fatalf("gzip read %q", b2)
	}
}
