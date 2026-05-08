package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ConnConfig holds connection settings shared by export/import and discovery.
type ConnConfig struct {
	Host    string
	Port    int
	Socket  string
	User    string
	Pass    string
	SSLMode string // disabled, preferred, skip-verify (mapped to driver), verify-ca, verify-identity

	ConnectTimeout time.Duration
}

func (c *ConnConfig) PasswordFromEnv() string {
	if c.Pass != "" {
		return c.Pass
	}
	if v := os.Getenv("DBMOV_PASSWORD"); v != "" {
		return v
	}
	return os.Getenv("MYSQL_PASSWORD")
}

func bindConnFlags(cmd *cobra.Command, cfg *ConnConfig) {
	cmd.PersistentFlags().StringVar(&cfg.Host, "host", "127.0.0.1", "MySQL/MariaDB host")
	cmd.PersistentFlags().IntVar(&cfg.Port, "port", 3306, "TCP port (ignored when --socket is set)")
	cmd.PersistentFlags().StringVar(&cfg.Socket, "socket", "", "Unix socket path (overrides host/port)")
	cmd.PersistentFlags().StringVar(&cfg.User, "user", "", "database user (required)")
	cmd.PersistentFlags().StringVar(&cfg.Pass, "password", "", "password (prefer DBMOV_PASSWORD or MYSQL_PASSWORD;)")
	cmd.PersistentFlags().StringVar(&cfg.SSLMode, "ssl-mode", "preferred", `TLS: disabled|preferred|skip-verify|required|verify-ca|verify-identity`)

	cmd.PersistentFlags().DurationVar(&cfg.ConnectTimeout, "connect-timeout", 30*time.Second, "initial connection timeout")
}

func requireUser(cmd *cobra.Command, cfg *ConnConfig) error {
	if strings.TrimSpace(cfg.User) == "" {
		return fmt.Errorf("%s: --user is required", cmd.Name())
	}
	return nil
}

// isMariaDBClient reports whether exe is a MariaDB mysql/mysqldump build (parses exe --version).
func isMariaDBClient(exe string) bool {
	exe = strings.TrimSpace(exe)
	if exe == "" {
		return false
	}
	out, err := exec.Command(exe, "--version").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), "mariadb")
}

// mysqlClientSSLCnf emits [client]-group SSL lines compatible with mysql or mariadb-cli.
func mysqlClientSSLCnf(sslMode string, mariaSyntax bool) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(sslMode))
	if mariaSyntax {
		switch mode {
		case "disabled", "false", "off":
			return "skip-ssl\n", nil
		case "preferred":
			// MariaDB has no ssl-mode=; omit so defaults (~SSL on where applicable) mirror driver "preferred".
			return "", nil
		case "skip-verify":
			return "ssl-verify-server-cert=false\n", nil
		case "required":
			return "ssl-verify-server-cert=true\n", nil
		case "verify-ca":
			return "ssl-verify-server-cert=true\n", nil
		case "verify-identity":
			return "ssl-verify-server-cert=true\n", nil
		default:
			return "", fmt.Errorf("unknown --ssl-mode %q", sslMode)
		}
	}
	var b strings.Builder
	switch mode {
	case "disabled", "false", "off":
		// leave unset; some clients treat as plain TCP
	case "preferred":
		fmt.Fprintf(&b, "ssl-mode=PREFERRED\n")
	case "skip-verify":
		fmt.Fprintf(&b, "ssl-mode=REQUIRED\n")
		// mysqldump/mysql 8+: skip verify via ssl-mode VERIFY_CA without ca is driver-specific;
		// client uses ssl-verify-server-cert=false when supported — not all versions; keep PREFERRED+ for max compat use skip-verify as REQUIRED minimum
		fmt.Fprintf(&b, "ssl-verify-server-cert=false\n")
	case "required":
		fmt.Fprintf(&b, "ssl-mode=REQUIRED\n")
	case "verify-ca":
		fmt.Fprintf(&b, "ssl-mode=VERIFY_CA\n")
	case "verify-identity":
		fmt.Fprintf(&b, "ssl-mode=VERIFY_IDENTITY\n")
	default:
		return "", fmt.Errorf("unknown --ssl-mode %q", sslMode)
	}
	return b.String(), nil
}

// WriteMySQLClientDefaults writes a temporary cnf fragment readable only by the owner.
// cliBinary is typically the mysql or mysqldump path (--mysql / --mysqldump): used only to emit
// MariaDB-compatible SSL options instead of oracle mysql ssl-mode= which MariaDB rejects.
// Caller must remove the file after use.
func WriteMySQLClientDefaults(cfg ConnConfig, pass string, cliBinary string) (path string, cleanup func(), err error) {
	dir := os.TempDir()
	f, err := os.CreateTemp(dir, "dbmov-my-*.cnf")
	if err != nil {
		return "", nil, err
	}
	path = f.Name()
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		os.Remove(path)
		return "", nil, err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "[client]\nuser=%s\npassword=%s\n", escapeCnfValue(cfg.User), escapeCnfValue(pass))
	if cfg.Socket != "" {
		fmt.Fprintf(&b, "socket=%s\n", escapeCnfValue(cfg.Socket))
	} else {
		// MySQL/MariaDB CLI treats host "localhost" as unix socket (e.g. /tmp/mysql.sock);
		// go-sql-driver uses TCP. Emit loopback IPv4 so mysqldump/mysql match discovery.
		h := tcpHostForMySQLCLI(cfg.Host)
		fmt.Fprintf(&b, "host=%s\nport=%d\n", escapeCnfValue(h), cfg.Port)
	}

	sslFrag, sslErr := mysqlClientSSLCnf(cfg.SSLMode, isMariaDBClient(cliBinary))
	if sslErr != nil {
		f.Close()
		os.Remove(path)
		return "", nil, sslErr
	}
	b.WriteString(sslFrag)

	if _, err := f.WriteString(b.String()); err != nil {
		f.Close()
		os.Remove(path)
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", nil, err
	}

	cleanup = func() { _ = os.Remove(path) }
	return path, cleanup, nil
}

func tcpHostForMySQLCLI(host string) string {
	if strings.EqualFold(strings.TrimSpace(host), "localhost") {
		return "127.0.0.1"
	}
	return host
}

func escapeCnfValue(s string) string {
	if !strings.ContainsAny(s, " \t\n\"'#;") {
		return s
	}
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return `"` + s + `"`
}

func absPath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("empty path")
	}
	return filepath.Abs(p)
}
