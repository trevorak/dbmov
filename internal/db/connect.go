package db

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Open opens a sql.DB using the built DSN.
func Open(dsnStr string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsnStr)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	return db, nil
}

// PingContext verifies connectivity.
func PingContext(ctx context.Context, db *sql.DB) error {
	return db.PingContext(ctx)
}

// ServerVersion returns SELECT VERSION() or an error.
func ServerVersion(ctx context.Context, db *sql.DB) (string, error) {
	var v string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&v); err != nil {
		return "", err
	}
	return strings.TrimSpace(v), nil
}

// BuildDSN builds a go-sql-driver/mysql DSN from fields.
func BuildDSN(host string, port int, socket, user, pass, sslMode string) (string, error) {
	cfg := mysql.NewConfig()
	cfg.User = user
	cfg.Passwd = pass
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(host, fmt.Sprintf("%d", port))
	cfg.Params = map[string]string{"parseTime": "true", "multiStatements": "true"}
	if socket != "" {
		cfg.Net = "unix"
		cfg.Addr = socket
	}
	tls, err := driverTLSConfigName(sslMode)
	if err != nil {
		return "", err
	}
	cfg.TLSConfig = tls
	return cfg.FormatDSN(), nil
}

func driverTLSConfigName(sslMode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(sslMode)) {
	case "disabled", "false", "off", "":
		return "", nil
	case "preferred":
		return "preferred", nil
	case "skip-verify":
		return "skip-verify", nil
	case "required", "verify-ca", "verify-identity":
		return "true", nil
	default:
		return "", fmt.Errorf("unknown ssl-mode %q", sslMode)
	}
}
