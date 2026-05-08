package main

import (
	"os"

	"github.com/trevor/dbmov/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
