//go:build !windows
// +build !windows

// undertow - a lightweight ssh server with a reverse connection feature
// Copyright (C) 2021  Ferdinor <ferdinor@mailbox.org>

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
)

// resolvePtyShell keeps the default /bin/bash from killing the session on
// bash-less targets: if the default is missing and /bin/sh exists, fall
// back to it. An explicit -s or RS_SHELL path is honoured verbatim.
func resolvePtyShell(shell string) string {
	if shell != defaultShell {
		return shell
	}
	if _, err := os.Stat(shell); err == nil {
		return shell
	}
	if _, err := os.Stat("/bin/sh"); err == nil {
		outInfof("%s not present on the target; falling back to /bin/sh", shell)
		return "/bin/sh"
	}
	return shell
}

func createPty(s ssh.Session, shell string) {
	var (
		ptyReq, winCh, _ = s.Pty()
		cmd              = exec.CommandContext(s.Context(), resolvePtyShell(shell))
	)

	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
	if currentUser, err := user.Current(); err == nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", currentUser.HomeDir))
	}
	f, err := pty.Start(cmd)
	if err != nil {
		outWarnf("Could not start shell: %v", err)
		io.WriteString(s, "Failed to start shell\n")
		s.Exit(255)
		return
	}
	defer f.Close()
	go func() {
		for win := range winCh {
			winSize := &pty.Winsize{Rows: uint16(win.Height), Cols: uint16(win.Width)}
			pty.Setsize(f, winSize)
		}
	}()

	go func() {
		io.Copy(f, s)
		s.Close()
	}()
	go func() {
		io.Copy(s, f)
		s.Close()
	}()

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			outWarnf("Session ended with error: %v", err)
			s.Exit(255)
			return
		}
		outOkf("Session ended normally")
		s.Exit(cmd.ProcessState.ExitCode())
		return

	case <-s.Context().Done():
		outInfof("Session terminated: %v", s.Context().Err())
		return
	}
}
