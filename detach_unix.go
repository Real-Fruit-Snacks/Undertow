//go:build !windows
// +build !windows

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
		Setsid: true,
	}
	// A detached payload must not hold the launching session (nc, winrm)
	// open through inherited handles: point all three stdio fds at null.
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
