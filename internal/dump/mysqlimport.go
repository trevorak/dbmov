package dump

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// RunMySQL feeds dumpFile to mysql stdin
func RunMySQL(ctx context.Context, defaultsFile, dumpFile, mysqlPath string, stderr io.Writer) error {
	bin := mysqlPath
	if bin == "" {
		bin = "mysql"
	}
	args := []string{"--defaults-file=" + defaultsFile}
	cmd := exec.CommandContext(ctx, bin, args...)
	f, err := os.Open(dumpFile)
	if err != nil {
		return err
	}
	defer f.Close()
	cmd.Stdin = f
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
