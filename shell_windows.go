package tig

import (
	"context"
	"os/exec"
	"syscall"
)

// shellCommand runs a command line through cmd.exe, passing the line as it
// is written (Go's own argument quoting would change it).
func shellCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd.exe /d /s /c "` + command + `"`}
	return cmd
}
