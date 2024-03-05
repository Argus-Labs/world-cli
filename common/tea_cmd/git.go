package tea_cmd

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"
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

	oldModuleName := "github.com/argus-labs/starter-game-template/cardinal"
	err = refactorModuleName(oldModuleName, filepath.Base(targetDir))
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

func refactorModuleName(oldModuleName, newModuleName string) error {
	cardinalDir := "cardinal"
	// Update import paths in all Go files
	err := filepath.Walk(cardinalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			return replaceInFile(path, oldModuleName, newModuleName)
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Update the go.mod file
	goModFilePath := filepath.Join(cardinalDir, "go.mod")

	return replaceInFile(goModFilePath, oldModuleName, newModuleName)
}

func replaceInFile(filePath, oldStr, newStr string) error {
	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, strings.ReplaceAll(scanner.Text(), oldStr, newStr))
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}
