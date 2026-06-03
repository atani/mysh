package cmd

import (
	"os"

	"golang.org/x/term"

	"github.com/atani/mysh/internal/crypto"
)

// readPassword reads a secret from the terminal with echo disabled. It is a
// package-level variable so tests can supply a deterministic value without a
// real TTY.
var readPassword = crypto.ReadPassword

// stdoutIsTTY and stdinIsTTY report whether the corresponding standard stream
// is connected to a terminal. They are package-level variables so tests can
// deterministically exercise TTY-dependent behavior (masking policy and the
// --raw production confirmation prompt) without an actual terminal.
var (
	stdoutIsTTY = func() bool { return term.IsTerminal(int(os.Stdout.Fd())) }
	stdinIsTTY  = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
)
