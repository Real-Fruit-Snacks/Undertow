//go:build windows
// +build windows

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
	"syscall"

	"github.com/ActiveState/termtest/conpty"
	"github.com/gliderlabs/ssh"
	"golang.org/x/sys/windows"
)

func createPty(s ssh.Session, shell string) {
	ptyReq, winCh, _ := s.Pty()
	vsn := windows.RtlGetVersion()

	if vsn.MajorVersion < 10 ||
		vsn.BuildNumber < 17763 {
		// Interactive Pty via ssh-shellhost.exe

		outWarnf("Windows version too old to support ConPTY shell")

		if shell == defaultShell {
			outWarnf("No fully interactive shell available, denying PTY request")
			io.WriteString(s, "No ConPTY shell or ssh-shellhost enhanced shell available. "+
				"Please append 'cmd' to your ssh command to gain shell access, i.e. "+
				"'ssh <OPTIONS> <IP> cmd'.\n")
			s.Exit(255)
			return
		}
		outInfof("Launching shell with ssh-shellhost.exe")

		cmd := exec.CommandContext(s.Context(), shell)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CmdLine:       " " + "---pty cmd", // Must leave a space to the beginning
			CreationFlags: 0x08000000,
		}
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

	} else {
		// Interactive ConPTY

		cpty, err := conpty.New(int16(ptyReq.Window.Width), int16(ptyReq.Window.Height))
		if err != nil {
			outWarnf("Could not open a conpty terminal: %v", err)
			io.WriteString(s, "Failed to create terminal\n")
			s.Exit(255)
			return
		}
		defer cpty.Close()

		// Dynamically handle resizes of terminal window
		go func() {
			for win := range winCh {
				cpty.Resize(uint16(win.Width), uint16(win.Height))
			}
		}()

		// Spawn and catch new powershell process
		pid, _, err := cpty.Spawn(
			"C:\\WINDOWS\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
			[]string{},
			&syscall.ProcAttr{
				Env: os.Environ(),
			},
		)
		if err != nil {
			outWarnf("Could not spawn a powershell: %v", err)
			io.WriteString(s, "Failed to spawn shell\n")
			s.Exit(255)
			return
		}
		outInfof("New process with pid %d spawned", pid)
		process, err := os.FindProcess(pid)
		if err != nil {
			outWarnf("Failed to find process: %v", err)
			io.WriteString(s, "Failed to find shell process\n")
			s.Exit(255)
			return
		}
		defer process.Kill()

		// Link data streams of ssh session and conpty
		go func() {
			io.Copy(s, cpty.OutPipe())
			s.Close()
		}()
		go func() {
			io.Copy(cpty.InPipe(), s)
			s.Close()
		}()

		done := make(chan struct {
			*os.ProcessState
			error
		}, 1)
		go func() {
			ps, err := process.Wait()
			done <- struct {
				*os.ProcessState
				error
			}{ps, err}
		}()

		select {
		case result := <-done:
			if result.error != nil {
				outWarnf("Error waiting for process: %v", result.error)
				s.Exit(255)
				return
			}
			outOkf("Session ended normally, exit code %d", result.ProcessState.ExitCode())
			s.Exit(result.ProcessState.ExitCode())
			return

		case <-s.Context().Done():
			outInfof("Session terminated: %v", s.Context().Err())
			return
		}
	}
}
