package db

import (
	"reflect"
	"testing"
)

func TestDriverTLSConfigName(t *testing.T) {
	tests := []struct {
		mode string
		want string
	}{
		{"disabled", ""},
		{"preferred", "preferred"},
		{"skip-verify", "skip-verify"},
		{"required", "true"},
	}
	for _, tt := range tests {
		got, err := driverTLSConfigName(tt.mode)
		if err != nil {
			t.Fatalf("driverTLSConfigName(%q): %v", tt.mode, err)
		}
		if got != tt.want {
			t.Fatalf("driverTLSConfigName(%q) = %q, want %q", tt.mode, got, tt.want)
		}
	}
	if _, err := driverTLSConfigName("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildDSN_TCPAndSocket(t *testing.T) {
	d, err := BuildDSN("h", 3307, "", "u", "p", "disabled")
	if err != nil {
		t.Fatal(err)
	}
	if d == "" {
		t.Fatal("empty dsn")
	}

	d2, err := BuildDSN("ignored", 0, "/tmp/mysql.sock", "u", "p", "disabled")
	if err != nil {
		t.Fatal(err)
	}
	if d2 == "" {
		t.Fatal("empty dsn")
	}
}

func TestFilterDiscoveredNames(t *testing.T) {
	raw := []string{"information_schema", "mysql", "performance_schema", "sys", "app", "app_test", "other"}

	got, err := FilterDiscoveredNames(raw, false, nil, []string{"app_*"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"app", "other"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}

	got, err = FilterDiscoveredNames(raw, false, []string{"app"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want = []string{"app"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("include: got %#v, want %#v", got, want)
	}

	got, err = FilterDiscoveredNames(raw, true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// still drops info/sys/perf; mysql included
	hasMysql := false
	for _, s := range got {
		if s == "mysql" {
			hasMysql = true
		}
	}
	if !hasMysql {
		t.Fatal("expected mysql when includeMySQL")
	}
}
