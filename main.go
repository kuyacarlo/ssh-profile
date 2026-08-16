package main

import (
	"github.com/kuyacarlo/ssh-profile/cmd"
)

// Set by -ldflags at build time.
var (
	Version     = "0.2.0"
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
