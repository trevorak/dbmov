package main

import (
	"os"

	"github.com/trevorak/dbmov/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
