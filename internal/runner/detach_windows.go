package runner

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// startDetached sobe `orion serve --no-browser` fora deste console: fechar o terminal não mata o painel.
func startDetached() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "serve", "--no-browser")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
