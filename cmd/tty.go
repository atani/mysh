package cmd

import (
	"os"
	"os/exec"

	"golang.org/x/term"

	"github.com/atani/mysh/internal/crypto"
)

// readPassword reads a secret from the terminal with echo disabled. It is a
// package-level variable so tests can supply a deterministic value without a
// real TTY.
var readPassword = crypto.ReadPassword

// execCommand builds an external client process (mysql/mycli). It is a
// package-level variable so tests can substitute a stub command and exercise
// the CLI output/capture paths without a real client binary or database.
var execCommand = exec.Command

// lookPath reports whether an executable exists in PATH. It is overridable for
// tests of the mycli/mysql client-selection logic.
var lookPath = exec.LookPath

// stdoutIsTTY and stdinIsTTY report whether the corresponding standard stream
// is connected to a terminal. They are package-level variables so tests can
// deterministically exercise TTY-dependent behavior (masking policy and the
// --raw production confirmation prompt) without an actual terminal.
var (
	stdoutIsTTY = func() bool { return term.IsTerminal(int(os.Stdout.Fd())) }
	stdinIsTTY  = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
)
