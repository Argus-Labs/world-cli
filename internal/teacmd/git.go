package teacmd

import (
	"strings"
)

type GitCloneFinishMsg struct {
	Err error
}

func (t *teaCmd) git(args ...string) (string, error) {
	output, err := t.terminal.Exec("git", args...)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (t *teaCmd) GitCloneCmd(url string, targetDir string, initMsg string) (err error) {
	_, err = t.git("clone", url, targetDir)
	if err != nil {
		return
	}

	err = t.terminal.Chdir(targetDir)
	if err != nil {
		return
	}

	rev, err := t.git("rev-list", "--tags", "--max-count=1")
	if err != nil {
		return
	}

	tag, err := t.git("describe", "--tags", strings.TrimSpace(rev))
	if err != nil {
		return
	}

	_, err = t.git("checkout", strings.TrimSpace(tag))
	if err != nil {
		return
	}

	err = t.terminal.Rm(".git")
	if err != nil {
		return
	}

	_, err = t.git("init")
	if err != nil {
		return
	}

	_, err = t.git("add", "-A")
	if err != nil {
		return
	}

	_, err = t.git("commit", "-m", initMsg)
	if err != nil {
		return
	}

	return
}
