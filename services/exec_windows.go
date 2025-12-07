//go:build windows
// +build windows

package services

import (
	"os/exec"
	"syscall"
)

// hideWindow sets the SysProcAttr to hide the command window on Windows
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
