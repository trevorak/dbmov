package dump

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type gzipReadCloser struct {
	f  *os.File
	gr *gzip.Reader
}

func (g *gzipReadCloser) Read(p []byte) (int, error) {
	return g.gr.Read(p)
}

func (g *gzipReadCloser) Close() error {
	err1 := g.gr.Close()
	err2 := g.f.Close()
	return errors.Join(err1, err2)
}

// OpenDumpReader opens a plain SQL dump or a gzip-compressed dump (.gz suffix on basename).
func OpenDumpReader(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	b := filepath.Base(path)
	if strings.HasSuffix(strings.ToLower(b), ".gz") {
		gr, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		return &gzipReadCloser{f: f, gr: gr}, nil
	}
	return f, nil
}

// RunMySQL feeds dumpFile to mysql stdin.
func RunMySQL(ctx context.Context, defaultsFile, dumpFile, mysqlPath string, stderr io.Writer) error {
	bin := mysqlPath
	if bin == "" {
		bin = "mysql"
	}
	args := []string{"--defaults-file=" + defaultsFile}
	cmd := exec.CommandContext(ctx, bin, args...)
	rc, err := OpenDumpReader(dumpFile)
	if err != nil {
		return err
	}
	defer rc.Close()
	cmd.Stdin = rc
	var errBuf bytes.Buffer
	if stderr != nil {
		cmd.Stderr = io.MultiWriter(&errBuf, stderr)
	} else {
		cmd.Stderr = &errBuf
	}
	cmd.Stdout = io.Discard
	if err := cmd.Run(); err != nil {
		msg := errBuf.String()
		if msg != "" {
			return fmt.Errorf("mysql restore %s: %w: %s", dumpFile, err, msg)
		}
		return fmt.Errorf("mysql restore %s: %w", dumpFile, err)
	}
	return nil
}
