package main

import (
	"github.com/ssh-profiles/git-ssh/cmd"
)

// Set by -ldflags at build time.
var (
	Version     = "0.1.0"
	CommitHash  = "Unknown"
	CompileDate = "Unknown"
)

func main() {
	root := cmd.New()
	root.Version = Version
	root.CommitHash = CommitHash
	root.CompileDate = CompileDate
	root.Execute()
}
