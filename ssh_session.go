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
	"errors"
	"io"
	"os"
	"os/exec"

	"github.com/gliderlabs/ssh"
)

func createSSHSessionHandler(shell string) ssh.Handler {
	return func(s ssh.Session) {
		outInfof("New login from %s@%s", s.User(), s.RemoteAddr().String())
		_, _, isPty := s.Pty()

		switch {
		case isPty:
			outInfof("PTY requested")

			createPty(s, shell)

		case len(s.Command()) > 0:
			outInfof("Command execution requested: '%s'", s.RawCommand())

			cmd := exec.CommandContext(s.Context(), s.Command()[0], s.Command()[1:]...)

			// We use StdinPipe to avoid blocking on missing input
			if stdin, err := cmd.StdinPipe(); err != nil {
				outWarnf("Could not initialize stdinPipe: %v", err)
				s.Exit(255)
				return
			} else {
				go func() {
					// Client EOF becomes the command's EOF. The session
					// lifecycle belongs to the select below — closing the
					// session here would race s.Exit and the client would
					// see no exit status.
					defer stdin.Close()
					if _, err := io.Copy(stdin, s); err != nil && !errors.Is(err, os.ErrClosed) {
						outWarnf("Error while copying input from %s to stdin: %v", s.RemoteAddr().String(), err)
					}
				}()
			}

			cmd.Stdout = s
			cmd.Stderr = s

			done := make(chan error, 1)
			go func() { done <- cmd.Run() }()

			select {
			case err := <-done:
				if err != nil {
					outWarnf("Command execution failed: %v", err)
					io.WriteString(s, "Command execution failed: "+err.Error()+"\n")
					s.Exit(255)
					return
				}
				outOkf("Command execution successful")
				s.Exit(cmd.ProcessState.ExitCode())
				return

			case <-s.Context().Done():
				outInfof("Session terminated: %v", s.Context().Err())
				return
			}

		default:
			outInfof("No PTY requested, no command supplied")

			// Keep this open until the session exits, could e.g. be port forwarding
			<-s.Context().Done()
			outInfof("Session terminated: %v", s.Context().Err())
			return
		}
	}
}
