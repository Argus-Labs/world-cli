package tea_cmd

import (
	"bytes"
	"github.com/magefile/mage/sh"
	"os"
	"strings"
)

type GitCloneFinishMsg struct {
	Err error
}

func git(args ...string) (string, error) {
	var outBuff, errBuff bytes.Buffer
	_, err := sh.Exec(nil, &outBuff, &errBuff, "git", args...)
	if err != nil {
		return "", err
	}
	return outBuff.String(), nil
}

func GitCloneCmd(url string, targetDir string, initMsg string) (err error) {
	_, err = git("clone", url, targetDir)
	if err != nil {
		return
	}

	err = os.Chdir(targetDir)
	if err != nil {
		return
	}

	rev, err := git("rev-list", "--tags", "--max-count=1")
	if err != nil {
		return
	}

	tag, err := git("describe", "--tags", strings.TrimSpace(rev))
	if err != nil {
		return
	}

	_, err = git("checkout", strings.TrimSpace(tag))
	if err != nil {
		return
	}

	err = os.RemoveAll(".git")
	if err != nil {
		return
	}

	_, err = git("init")
	if err != nil {
		return
	}

	_, err = git("add", "-A")
	if err != nil {
		return
	}

	_, err = git("commit", "-m", initMsg)
	if err != nil {
		return
	}

	return
}
