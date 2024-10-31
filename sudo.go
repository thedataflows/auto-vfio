package main

import (
	"fmt"
	"os"
	"os/exec"
)

// Checks if the user has passwordless sudo access
func hasPasswordlessSudo() bool {
	cmd := exec.Command("sudo", "-n", "true")
	err := cmd.Run()
	return err == nil
}

// Re-run the current executable with sudo
func reRunElevated() error {
	if os.Geteuid() == 0 {
		return nil
	}

	if !hasPasswordlessSudo() {
		fmt.Println("This operation requires elevated privileges. Requesting sudo...")
	}
	// Get the path to the current executable
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Prepare sudo command with all original arguments
	cmd := exec.Command("sudo", append([]string{exe}, os.Args[1:]...)...) //nolint:gosec

	// Connect standard IO
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run and wait for completion
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("sudo execution failed: %w", err)
	}

	// Exit the original process after sudo execution
	os.Exit(0)
	return nil
}
