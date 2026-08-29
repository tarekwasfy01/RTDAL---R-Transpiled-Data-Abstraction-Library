package gui

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed rtdal_gui.ps1
var toolbox []byte

func Show(executable string) error {
	path := filepath.Join(os.TempDir(), "RTDAL-Toolbox.ps1")
	if err := os.WriteFile(path, toolbox, 0600); err != nil {
		return err
	}
	out, err := exec.Command("powershell.exe", "-NoProfile", "-STA", "-ExecutionPolicy", "Bypass", "-File", path, "-RTDALPath", executable).CombinedOutput()
	if err != nil {
		return fmt.Errorf("toolbox: %w: %s", err, string(out))
	}
	return nil
}
