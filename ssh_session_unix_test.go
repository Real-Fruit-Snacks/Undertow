//go:build !windows
// +build !windows

package main

import (
	"os"
	"testing"
)

func TestResolvePtyShell(t *testing.T) {
	// An explicit -s or RS_SHELL path is honoured verbatim, even a
	// nonexistent one: the operator asked for that exact shell.
	if got := resolvePtyShell("/custom/shell"); got != "/custom/shell" {
		t.Fatalf("explicit shell = %q, want /custom/shell", got)
	}

	old := defaultShell
	defer func() { defaultShell = old }()

	// A present default is used as-is.
	defaultShell = "/bin/sh"
	if got := resolvePtyShell(defaultShell); got != "/bin/sh" {
		t.Fatalf("present default = %q, want /bin/sh", got)
	}

	// A missing default falls back to /bin/sh when the target has it.
	defaultShell = "/nonexistent/bash"
	if _, err := os.Stat("/bin/sh"); err == nil {
		if got := resolvePtyShell(defaultShell); got != "/bin/sh" {
			t.Fatalf("missing default = %q, want /bin/sh", got)
		}
	}
}
