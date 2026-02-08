package main

import (
	"github.com/saeedalam/teamcontext/internal/cli"
)

// Set by ldflags at build time
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cli.SetVersionInfo(version, commit, date)
	cli.Execute()
}
