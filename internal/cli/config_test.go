package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTCPHostForMySQLCLI_localhostUsesLoopbackIPv4(t *testing.T) {
	for _, h := range []string{"localhost", "LOCALHOST", " localhost "} {
		if got := tcpHostForMySQLCLI(h); got != "127.0.0.1" {
			t.Fatalf("tcpHostForMySQLCLI(%q) = %q", h, got)
		}
	}
	if tcpHostForMySQLCLI("db.internal") != "db.internal" {
		t.Fatal("non-localhost unchanged")
	}
}

func TestMysqlClientSSLCnf_MariaVersusOraclePreferred(t *testing.T) {
	maria, err := mysqlClientSSLCnf("preferred", true)
	if err != nil || maria != "" {
		t.Fatalf("mariadb preferred: got %q err %v", maria, err)
	}
	ora, err := mysqlClientSSLCnf("preferred", false)
	if err != nil || ora != "ssl-mode=PREFERRED\n" {
		t.Fatalf("oracle preferred: got %q err %v", ora, err)
	}
}

func TestMysqlClientSSLCnf_MariaDisabled(t *testing.T) {
	s, err := mysqlClientSSLCnf("disabled", true)
	if err != nil || s != "skip-ssl\n" {
		t.Fatalf("got %q err %v", s, err)
	}
}

func TestMysqlClientSSLCnf_MariaSkipVerify(t *testing.T) {
	s, err := mysqlClientSSLCnf("skip-verify", true)
	if err != nil || s != "ssl-verify-server-cert=false\n" {
		t.Fatalf("got %q err %v", s, err)
	}
}

func TestIsMariaDBClient(t *testing.T) {
	if isMariaDBClient("/nonexistent/dbmov-no-such-mysql-binary") {
		t.Fatal("expected false for missing binary")
	}
	if isMariaDBClient("") {
		t.Fatal("empty path false")
	}
	dir := t.TempDir()
	shim := filepath.Join(dir, "fake-mysql")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\necho 'mysql  Ver 15.2 for debian Distrib 11.8-MariaDB'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !isMariaDBClient(shim) {
		t.Fatal("expected Mariadb substring match")
	}
}
