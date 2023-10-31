package tea_cmd

import (
	"bytes"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/magefile/mage/sh"
	"os"
)

type GitCloneFinishMsg struct {
	Err error
}

func git(args ...string) error {
	var outBuff, errBuff bytes.Buffer
	_, err := sh.Exec(nil, &outBuff, &errBuff, "git", args...)
	if err != nil {
		return err
	}
	return nil
}

func GitCloneCmd(url string, targetDir string, initMsg string) tea.Cmd {
	return func() tea.Msg {
		err := git("clone", url, targetDir)
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

		err = git("init")
		if err != nil {
			return GitCloneFinishMsg{Err: err}
		}

		err = git("add", "-A")
		if err != nil {
			return GitCloneFinishMsg{Err: err}
		}

		err = git("commit", "-m", initMsg)
		if err != nil {
			return GitCloneFinishMsg{Err: err}
		}

		return GitCloneFinishMsg{Err: nil}
	}
}
