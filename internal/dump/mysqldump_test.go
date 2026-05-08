package dump

import (
	"reflect"
	"strings"
	"testing"
)

func TestMysqldumpArgs(t *testing.T) {
	o := MysqldumpOptions{SetGTIDPurged: "OFF", ColumnStatisticsFalse: true}
	args := MysqldumpArgs("/tmp/cnf", "mydb", "/out/dump.sql", o)
	if !strings.HasPrefix(args[0], "--defaults-file=/tmp/cnf") {
		t.Fatalf("first arg must be --defaults-file=..., got %v", args)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--defaults-file=/tmp/cnf") {
		t.Fatal(joined)
	}
	if !strings.Contains(joined, "--databases") || !strings.Contains(joined, "mydb") {
		t.Fatal(joined)
	}
	if !strings.Contains(joined, "--column-statistics=0") {
		t.Fatal(joined)
	}
	if !strings.Contains(joined, "--set-gtid-purged=OFF") {
		t.Fatal(joined)
	}
	if !strings.Contains(joined, "--result-file=") || !strings.Contains(joined, "dump.sql") {
		t.Fatal(joined)
	}

	o2 := MysqldumpOptions{ColumnStatisticsFalse: false}
	args2 := MysqldumpArgs("/cnf", "a", "/r.sql", o2)
	for _, a := range args2 {
		if a == "--column-statistics=0" {
			t.Fatal("should omit column-statistics")
		}
	}
}

func TestMysqldumpArgsOrder(t *testing.T) {
	args := MysqldumpArgs("DEF", "db", "OUT", MysqldumpOptions{})
	want := "--defaults-file=DEF"
	if !reflect.DeepEqual(args[0], want) {
		t.Fatalf(`first arg must be %q (only option file read), got %v`, want, args)
	}
}
