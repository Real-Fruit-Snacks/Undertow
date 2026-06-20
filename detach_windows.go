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
	return cmd.Start()
}
