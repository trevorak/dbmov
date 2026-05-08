package cli

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trevor/dbmov/internal/output"

	"github.com/trevor/dbmov/internal/db"
	"github.com/trevor/dbmov/internal/dump"
	"github.com/trevor/dbmov/internal/manifest"
)

func init() {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Restore databases from a backup directory produced by dbmov export",
		RunE:  runImport,
	}
	cmd.Flags().StringVar(&impFrom, "from", "", "backup directory containing manifest.json and *.sql (required)")
	cmd.Flags().StringVar(&impMysql, "mysql", "mysql", "path to mysql client binary")
	cmd.Flags().BoolVar(&impContinueOnError, "continue-on-error", false, "keep restoring after a failure")
	rootCmd.AddCommand(cmd)
}

var (
	impFrom            string
	impMysql           string
	impContinueOnError bool
)

func runImport(cmd *cobra.Command, _ []string) error {
	if err := requireUser(cmd, &conn); err != nil {
		return err
	}
	if strings.TrimSpace(impFrom) == "" {
		return output.Errorf("import --from is required")
	}
	fromDir, err := absPath(impFrom)
	if err != nil {
		return err
	}

	type job struct {
		dbName string
		file   string
	}

	var jobs []job
	if m, err := manifest.ReadOrNil(fromDir); err != nil {
		return err
	} else if m != nil && len(m.Entries) > 0 {
		// Prefer successful exports; still allow retry of failed lines if user edits manifest.
		for _, e := range m.Entries {
			if strings.TrimSpace(e.File) == "" {
				continue
			}
			p := filepath.Join(fromDir, e.File)
			jobs = append(jobs, job{dbName: e.Database, file: p})
		}
	}
	if len(jobs) == 0 {
		matches, err := filepath.Glob(filepath.Join(fromDir, "*.sql"))
		if err != nil {
			return err
		}
		sort.Strings(matches)
		for _, p := range matches {
			base := filepath.Base(p)
			name := strings.TrimSuffix(base, ".sql")
			jobs = append(jobs, job{dbName: name, file: p})
		}
	}
	if len(jobs) == 0 {
		return output.Errorf("no manifest entries and no *.sql files in %s", fromDir)
	}

	w := cmd.OutOrStdout()
	output.Fprintf(w, "dbmov import: restoring %d database(s) from %s\n\n", len(jobs), fromDir)

	pass := conn.PasswordFromEnv()
	dsnStr, err := db.BuildDSN(conn.Host, conn.Port, conn.Socket, conn.User, pass, conn.SSLMode)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), conn.ConnectTimeout)
	sqldb, err := db.Open(dsnStr)
	if err != nil {
		cancel()
		return err
	}
	defer sqldb.Close()
	if err := db.PingContext(ctx, sqldb); err != nil {
		cancel()
		return output.Errorf("ping: %w", err)
	}
	cancel()

	cnfPath, cleanup, err := WriteMySQLClientDefaults(conn, pass, impMysql)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx2 := cmd.Context()
	var hadErr bool
	var nOK int
	for i, j := range jobs {
		rel, relErr := filepath.Rel(fromDir, j.file)
		if relErr != nil {
			rel = filepath.Base(j.file)
		}

		output.Fprintf(w, "[%d/%d] restoring %s ...\n", i+1, len(jobs), j.dbName)

		if err := dump.RunMySQL(ctx2, cnfPath, j.file, impMysql, os.Stderr); err != nil {
			hadErr = true
			output.FprintTabbedf(w, "failed\n")
			output.Fprintf(os.Stderr, "dbmov import: %v\n", err)
			if !impContinueOnError {
				output.Fprintf(w, "\ndbmov import: stopped after failure on %s (%d restored before stop)\n", j.dbName, nOK)
				return err
			}
		} else {
			nOK++
			output.FprintTabbedf(w, "restored %s\n", rel)
		}
	}
	if hadErr {
		output.Fprintf(w, "\ndbmov import: finished with errors (%d ok of %d)\n", nOK, len(jobs))
		return output.Errorf("import finished with one or more errors")
	}
	output.Fprintf(w, "\ndbmov import: done (%d database(s))\n", nOK)
	return nil
}
