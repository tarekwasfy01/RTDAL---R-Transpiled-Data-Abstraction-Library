//go:build !windows

package gui

import "fmt"

func Show(executable string) error {
	return fmt.Errorf("the graphical toolbox requires Windows; use %s help for CLI commands", executable)
}
