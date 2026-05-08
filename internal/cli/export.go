package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/trevor/dbmov/internal/output"

	"github.com/trevor/dbmov/internal/db"
	"github.com/trevor/dbmov/internal/dump"
	"github.com/trevor/dbmov/internal/manifest"
	"github.com/trevor/dbmov/internal/version"
)

func init() {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Dump every matching database to SQL files under --out",
		RunE:  runExport,
	}
	cmd.Flags().StringVar(&exportOut, "to", "", "backup directory (required)")
	cmd.Flags().BoolVar(&exportIncludeMySQL, "include-mysql", false, "include the mysql system schema in discovery")
	cmd.Flags().StringSliceVar(&exportInclude, "include", nil, "optional glob; if set, only databases matching at least one pattern")
	cmd.Flags().StringSliceVar(&exportExclude, "exclude", nil, "glob patterns to exclude (e.g. test_*)")
	cmd.Flags().BoolVar(&exportContinueOnError, "continue-on-error", false, "keep dumping after a failure")
	cmd.Flags().StringVar(&exportMysqldump, "mysqldump", "mysqldump", "path to mysqldump binary")
	cmd.Flags().DurationVar(&exportDumpTimeout, "dump-timeout", 0, "per-database mysqldump timeout (0 = no limit)")
	cmd.Flags().StringVar(&exportGTIDPurged, "set-gtid-purged", "OFF", "passed to mysqldump --set-gtid-purged")
	cmd.Flags().BoolVar(&exportDumpColumnStatsOff, "dump-column-statistics-off", false, `pass mysqldump --column-statistics=0 (MySQL 8+ client dumping older servers; errors on mysqldump that lacks this flag)`)
	rootCmd.AddCommand(cmd)
}

var (
	exportOut                string
	exportIncludeMySQL       bool
	exportInclude            []string
	exportExclude            []string
	exportContinueOnError    bool
	exportMysqldump          string
	exportDumpTimeout        time.Duration
	exportGTIDPurged         string
	exportDumpColumnStatsOff bool
)

func runExport(cmd *cobra.Command, _ []string) error {
	if err := requireUser(cmd, &conn); err != nil {
		return err
	}
	if strings.TrimSpace(exportOut) == "" {
		return output.Errorf("export --to is required")
	}
	outDir, err := absPath(exportOut)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

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

	ctx2 := cmd.Context()
	sver, err := db.ServerVersion(ctx2, sqldb)
	if err != nil {
		return output.Errorf("server version: %w", err)
	}

	names, err := db.DiscoverDatabases(ctx2, sqldb, exportIncludeMySQL, exportInclude, exportExclude)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		output.Fprintln(os.Stderr, "dbmov export: no databases matched filters")
		return nil
	}

	w := cmd.OutOrStdout()
	output.Fprintf(w, "dbmov export: dumping %d database(s) to %s\n\n", len(names), outDir)

	cnfPath, cleanup, err := WriteMySQLClientDefaults(conn, pass, exportMysqldump)
	if err != nil {
		return err
	}
	defer cleanup()

	opts := dump.MysqldumpOptions{
		SetGTIDPurged:         exportGTIDPurged,
		ColumnStatisticsFalse: exportDumpColumnStatsOff,
		MysqldumpPath:         exportMysqldump,
	}

	m := &manifest.Manifest{
		ToolVersion:   version.Version,
		CreatedAt:     time.Now().UTC(),
		ServerVersion: sver,
		Entries:       nil,
	}

	var hadErr bool
	var nOK int

	for i, dbName := range names {
		sqlFile := filepath.Join(outDir, safeFileName(dbName)+".sql")
		rel, relErr := filepath.Rel(outDir, sqlFile)
		if relErr != nil {
			rel = filepath.Base(sqlFile)
		}

		output.Fprintf(w, "[%d/%d] dumping %s ...\n", i+1, len(names), dbName)

		dumpCtx := ctx2
		var dumpCancel context.CancelFunc
		if exportDumpTimeout > 0 {
			dumpCtx, dumpCancel = context.WithTimeout(ctx2, exportDumpTimeout)
		}
		err := dump.RunMysqldump(dumpCtx, cnfPath, dbName, sqlFile, opts, os.Stderr)
		if dumpCancel != nil {
			dumpCancel()
		}

		ent := manifest.Entry{Database: dbName, File: rel, ExitCode: 0}
		if err != nil {
			hadErr = true
			ent.ExitCode = 1
			ent.Error = err.Error()
			output.Fprintf(w, "         failed\n")
			output.Fprintf(os.Stderr, "dbmov export: %v\n", err)
			if !exportContinueOnError {
				m.Entries = append(m.Entries, ent)
				_ = manifest.Write(filepath.Join(outDir, manifest.FileName), m)
				output.Fprintf(w, "\ndbmov export: stopped after failure on %s (%d exported before stop)\n", dbName, nOK)
				return output.Errorf("export stopped after failure on %s", dbName)
			}
		} else {
			nOK++
			output.Fprintf(w, "         wrote %s\n", rel)
		}
		m.Entries = append(m.Entries, ent)
	}

	if err := manifest.Write(filepath.Join(outDir, manifest.FileName), m); err != nil {
		return err
	}
	output.Fprintf(w, "\ndbmov export: manifest.json written under %s\n", outDir)
	if hadErr {
		output.Fprintf(w, "dbmov export: finished with errors (%d ok of %d)\n", nOK, len(names))
		return output.Errorf("export finished with one or more errors")
	}
	output.Fprintf(w, "dbmov export: done (%d database(s))\n", nOK)
	return nil
}

func safeFileName(db string) string {
	var b strings.Builder
	for _, r := range db {
		switch r {
		case '/', '\\', ':', '\x00':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	s := b.String()
	if s == "" || s == "." || s == ".." {
		return "db"
	}
	return s
}
