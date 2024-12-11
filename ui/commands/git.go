package commands

import (
	"fmt"
	"os"
	"os/exec"
)

// GitCloneFinishMsg represents a message indicating git clone operation completion
type GitCloneFinishMsg struct {
	Err error
}

// GitCloneCmd clones a git repository and initializes it
func GitCloneCmd(url, dir, message string) error {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Change to the directory
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	// Initialize git repository
	initCmd := exec.Command("git", "init")
	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Add remote origin
	remoteCmd := exec.Command("git", "remote", "add", "origin", url)
	if err := remoteCmd.Run(); err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	// Pull from remote
	pullCmd := exec.Command("git", "pull", "origin", "main")
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("failed to pull from remote: %w", err)
	}

	// Add all files
	addCmd := exec.Command("git", "add", ".")
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	// Commit changes
	commitCmd := exec.Command("git", "commit", "-m", message)
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}
