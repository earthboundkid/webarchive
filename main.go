package main

import (
	"os"

	"github.com/carlmjohnson/exitcode"
	"github.com/carlmjohnson/webarchive/archive"
)

func main() {
	exitcode.Exit(archive.CLI(os.Args[1:]))
}
