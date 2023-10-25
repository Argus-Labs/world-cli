package tea_cmd

import (
	"bytes"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"os/exec"
)

type GitCloneFinishMsg struct {
	ErrBuf *bytes.Buffer
	Err    error
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
		cmd := exec.Command("git", "clone", url, targetDir)
		errBuf, err := Run(cmd)
		if err != nil {
			return GitCloneFinishMsg{ErrBuf: errBuf, Err: err}
		}

		err = os.Chdir(targetDir)
		if err != nil {
			return GitCloneFinishMsg{ErrBuf: nil, Err: err}
		}

		cmd = exec.Command("rm", "-rf", ".git")
		errBuf, err = Run(cmd)
		if err != nil {
			return GitCloneFinishMsg{ErrBuf: errBuf, Err: err}
		}

		cmd = exec.Command("git", "init")
		errBuf, err = Run(cmd)
		if err != nil {
			return GitCloneFinishMsg{ErrBuf: errBuf, Err: err}
		}

		cmd = exec.Command("git", "add", "-A")
		errBuf, err = Run(cmd)
		if err != nil {
			return GitCloneFinishMsg{ErrBuf: errBuf, Err: err}
		}

		cmd = exec.Command("git", "commit", "-m", initMsg)
		errBuf, err = Run(cmd)
		if err != nil {
			return GitCloneFinishMsg{ErrBuf: errBuf, Err: err}
		}

		return GitCloneFinishMsg{ErrBuf: nil, Err: nil}
	}
}
