package git

import (
	"os"
	"os/exec"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/config"
)

// CloneRepo clones a git repository to the specified directory
func CloneRepo(url, dir, message string) error {
	if err := os.MkdirAll(dir, config.GitDirPerm); err != nil {
		return eris.Wrapf(err, "failed to create directory %s", dir)
	}

	cmd := exec.Command("git", "clone", url, dir)
	if err := cmd.Run(); err != nil {
		return eris.Wrapf(err, "failed to clone repository %s", url)
	}

	if message != "" {
		cmd = exec.Command("git", "-C", dir, "commit", "--allow-empty", "-m", message)
		if err := cmd.Run(); err != nil {
			return eris.Wrapf(err, "failed to create initial commit")
		}
	}

	return nil
}
