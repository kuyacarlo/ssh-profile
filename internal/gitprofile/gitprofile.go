package gitprofile

import (
	"io"
	"os"
	"os/exec"
)

// Runner talks to the separate git-profile binary. It never couples to
// git-profile's config format — it only detects it and hands off control.
type Runner interface {
	// Available reports whether git-profile is on PATH.
	Available() bool
	// RunAdd starts interactive `git-profile add` with stdio passthrough.
	RunAdd(in io.Reader, out, errOut io.Writer) error
	// RunUse applies the same-named git-profile profile in the current repo.
	RunUse(name string, in io.Reader, out, errOut io.Writer) error
}

// Linker is the default Runner backed by exec.LookPath / exec.Command.
type Linker struct {
	lookPath func(string) (string, error)
}

// New returns a Linker using the real environment.
func New() *Linker {
	return &Linker{lookPath: exec.LookPath}
}

func (l *Linker) Available() bool {
	_, err := l.lookPath("git-profile")
	return err == nil
}

func (l *Linker) RunAdd(in io.Reader, out, errOut io.Writer) error {
	return l.run([]string{"add"}, in, out, errOut)
}

func (l *Linker) RunUse(name string, in io.Reader, out, errOut io.Writer) error {
	return l.run([]string{"use", name}, in, out, errOut)
}

func (l *Linker) run(args []string, in io.Reader, out, errOut io.Writer) error {
	cmd := exec.Command("git-profile", args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}

// Compile-time check that Linker satisfies Runner.
var _ Runner = (*Linker)(nil)

// Skip is a Runner that reports git-profile as unavailable. Useful for tests
// and for build environments where the companion binary must not be probed.
type Skip struct{}

func (Skip) Available() bool { return false }

func (Skip) RunAdd(_ io.Reader, _, _ io.Writer) error {
	return os.ErrNotExist
}

func (Skip) RunUse(_ string, _ io.Reader, _, _ io.Writer) error {
	return os.ErrNotExist
}

var _ Runner = Skip{}
