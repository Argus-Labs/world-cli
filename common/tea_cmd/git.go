package tea_cmd

import (
	"bytes"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/magefile/mage/sh"
	"os"
	"os/exec"
)

type GitCloneFinishMsg struct {
	Err error
}

func Run(cmd *exec.Cmd) (*bytes.Buffer, error) {
	var outBuff, errBuff bytes.Buffer
	cmd.Stdout = &outBuff
	cmd.Stderr = &errBuff

	err := cmd.Run()
	if err != nil {
		return &errBuff, err
	}

	return nil, nil
}

func GitCloneCmd(url string, targetDir string, initMsg string) tea.Cmd {
	return func() tea.Msg {
		err := sh.Run("git", "clone", url, targetDir)
		if err != nil {
			return GitCloneFinishMsg{Err: err}
		}

		err = os.Chdir(targetDir)
		if err != nil {
			return GitCloneFinishMsg{Err: err}
		}

		err = sh.Run("rm", "-rf", ".git")
		if err != nil {
			return GitCloneFinishMsg{Err: err}
		}

		err = sh.Run("git", "init")
		if err != nil {
			return GitCloneFinishMsg{Err: err}
		}

		err = sh.Run("git", "add", "-A")
		if err != nil {
			return GitCloneFinishMsg{Err: err}
		}

		err = sh.Run("git", "commit", "-m", initMsg)
		if err != nil {
			return GitCloneFinishMsg{Err: err}
		}

		return GitCloneFinishMsg{Err: nil}
	}
}
