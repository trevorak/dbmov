package dump

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// MysqldumpOptions configures mysqldump.
type MysqldumpOptions struct {
	SetGTIDPurged         string // OFF, ON, AUTO
	ColumnStatisticsFalse bool   // pass mysqldump --column-statistics=0 (MySQL 8+ client only)
	MysqldumpPath         string // default "mysqldump"
	Compress              bool   // streams mysqldump stdout through gzip into resultFile
}

// mysqldumpArgsBase builds mysqldump argv without --result-file (for stdout streaming).
func mysqldumpArgsBase(defaultsFile, database string, o MysqldumpOptions) []string {
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
	}
	if o.ColumnStatisticsFalse {
		args = append(args, "--column-statistics=0")
	}
	return args
}

// MysqldumpArgs builds the argument list.
// defaultsFile is a temp option file; passed as --defaults-file so it is the only file read.
// If resultFile is empty, --result-file is omitted (stdout mode).
func MysqldumpArgs(defaultsFile, database, resultFile string, o MysqldumpOptions) []string {
	args := mysqldumpArgsBase(defaultsFile, database, o)
	if resultFile != "" {
		args = append(args, "--result-file="+resultFile)
	}
	return args
}

// RunMysqldump executes mysqldump; stderr is captured and returned on failure.
// When o.Compress is true, streams stdout through gzip into resultFile (omit --result-file).
func RunMysqldump(ctx context.Context, defaultsFile, database, resultFile string, o MysqldumpOptions, stderr io.Writer) error {
	bin := o.MysqldumpPath
	if bin == "" {
		bin = "mysqldump"
	}
	var errBuf bytes.Buffer
	cmdStderr := io.Writer(&errBuf)
	if stderr != nil {
		cmdStderr = io.MultiWriter(&errBuf, stderr)
	}

	if o.Compress {
		args := mysqldumpArgsBase(defaultsFile, database, o)
		cmd := exec.CommandContext(ctx, bin, args...)
		cmd.Stderr = cmdStderr

		out, err := os.Create(resultFile)
		if err != nil {
			return fmt.Errorf("mysqldump %s: create output: %w", database, err)
		}
		gw := gzip.NewWriter(out)
		cmd.Stdout = gw

		runErr := cmd.Run()
		gzErr := gw.Close()
		outErr := out.Close()
		if runErr != nil {
			msg := errBuf.String()
			if msg != "" {
				return fmt.Errorf("mysqldump %s: %w: %s", database, runErr, msg)
			}
			return fmt.Errorf("mysqldump %s: %w", database, runErr)
		}
		if gzErr != nil {
			return fmt.Errorf("mysqldump %s: gzip: %w", database, gzErr)
		}
		if outErr != nil {
			return fmt.Errorf("mysqldump %s: close output: %w", database, outErr)
		}
		return nil
	}

	args := MysqldumpArgs(defaultsFile, database, resultFile, o)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stderr = cmdStderr
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
