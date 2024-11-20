//go:build !windows

package meh

import "syscall"

func procAttrs(cmd []string) *syscall.SysProcAttr {
	return nil
}
