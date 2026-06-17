package passthrough

import (
	"fmt"
	"os"
	"os/exec"
)

var ghBin string

var lookPath = exec.LookPath

func init() {
	ghBin, _ = lookPath("gh")
}

func IsGHAvailable() bool {
	return ghBin != ""
}

func GHPath() string {
	return ghBin
}

func Execute(args []string) error {
	if ghBin == "" {
		return fmt.Errorf("gh (GitHub CLI) not found in PATH")
	}

	cmd := exec.Command(ghBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("execute gh: %w", err)
	}
	return nil
}

func ExecuteWithOutput(args []string) (string, error) {
	if ghBin == "" {
		return "", fmt.Errorf("gh (GitHub CLI) not found in PATH")
	}

	cmd := exec.Command(ghBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("execute gh: %w", err)
	}
	return string(output), nil
}

func IsPassthroughCommand(cmd string) bool {
	knownCommands := map[string]bool{
		"commit": true,
		"pr":     true,
		"init":   true,
		"config": true,
		"why":    true,
		"log":    true,
		"diff":   true,
		"audit":  true,
	}

	if knownCommands[cmd] {
		return false
	}
	return true
}


