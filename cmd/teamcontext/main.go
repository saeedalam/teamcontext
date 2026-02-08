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
	println("[DEBUG] main: starting...")
	cli.SetVersionInfo(version, commit, date)
	cli.Execute()
}
