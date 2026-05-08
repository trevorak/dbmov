package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/trevorak/dbmov/internal/output"

	"github.com/trevorak/dbmov/internal/version"
)

var conn ConnConfig

var rootCmd = &cobra.Command{
	Use:   "dbmov",
	Short: "Export and import MySQL/MariaDB databases visible to a user",
}

// Execute runs the CLI and returns an exit code (0 on success).
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		output.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.Version = version.Version
	bindConnFlags(rootCmd, &conn)
}
