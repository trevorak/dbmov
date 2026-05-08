package dump

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
)

// MysqldumpOptions configures mysqldump invocation.
type MysqldumpOptions struct {
	SetGTIDPurged         string // e.g. OFF, ON, AUTO
	ColumnStatisticsFalse bool   // pass mysqldump --column-statistics=0 (MySQL 8+ client only)
	MysqldumpPath         string // default "mysqldump"
}

// MysqldumpArgs builds the argument list (for tests and execution).
// defaultsFile is a temp option file; passed as --defaults-file so it is the only file read
// (avoids merged [mysqldump] options from /etc and ~ that can break cross-flavour clients).
func MysqldumpArgs(defaultsFile, database, resultFile string, o MysqldumpOptions) []string {
	gtid := o.SetGTIDPurged
	if gtid == "" {
		gtid = "OFF"
	}
	args := []string{
		"--defaults-file=" + defaultsFile,
		"--single-transaction",
		"--routines",
		"--triggers",
		"--events",
		"--set-gtid-purged=" + gtid,
		"--databases",
		database,
		"--result-file=" + resultFile,
	}
	if o.ColumnStatisticsFalse {
		args = append(args, "--column-statistics=0")
	}
	return args
}

// RunMysqldump executes mysqldump; stderr is captured and returned on failure.
func RunMysqldump(ctx context.Context, defaultsFile, database, resultFile string, o MysqldumpOptions, stderr io.Writer) error {
	bin := o.MysqldumpPath
	if bin == "" {
		bin = "mysqldump"
	}
	args := MysqldumpArgs(defaultsFile, database, resultFile, o)
	cmd := exec.CommandContext(ctx, bin, args...)
	var errBuf bytes.Buffer
	if stderr != nil {
		cmd.Stderr = io.MultiWriter(&errBuf, stderr)
	} else {
		cmd.Stderr = &errBuf
	}
	cmd.Stdout = nil
	if err := cmd.Run(); err != nil {
		msg := errBuf.String()
		if msg != "" {
			return fmt.Errorf("mysqldump %s: %w: %s", database, err, msg)
		}
		return fmt.Errorf("mysqldump %s: %w", database, err)
	}
	return nil
}
