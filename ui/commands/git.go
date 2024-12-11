package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rotisserie/eris"
)

// GitCloneFinishMsg represents the result of a git clone operation
type GitCloneFinishMsg struct {
	Err error
}

// GitCloneCommand clones a repository
func GitCloneCommand(url, dir string) error {
	cmd := exec.Command("git", "clone", url, dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Initialize git and create initial commit
	if err := os.Chdir(dir); err != nil {
		return eris.Wrap(err, "failed to change directory")
	}

	return nil
}
