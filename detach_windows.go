//go:build windows
// +build windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func detachProcess() error {
	args := removeDFlag(os.Args[1:])
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), detachEnvKey+"=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x00000008 | 0x08000000, // DETACHED_PROCESS | CREATE_NO_WINDOW
	}
	// Same contract as unix: no inherited handles back to the session
	// that launched the payload.
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	return cmd.Start()
}
