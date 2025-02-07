package teacmd

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/rotisserie/eris"
)

const (
	oldModuleName = "github.com/argus-labs/starter-game-template/cardinal"

	tomlFile = "world.toml"

	routerKey = "ROUTER_KEY"

	tomlSectionCommon = "common"

	routerKeyLength = 32
)

type GitCloneFinishMsg struct {
	Err error
}

func git(args ...string) (string, error) {
	var outBuff, errBuff bytes.Buffer

	// Define environment variables
	env := map[string]string{
		"GIT_COMMITTER_NAME":  "World CLI",
		"GIT_COMMITTER_EMAIL": "no-reply@world.dev",
	}

	_, err := sh.Exec(env, &outBuff, &errBuff, "git", args...)
	if err != nil {
		return "", eris.Errorf("git %v failed: %s", args, errBuff.String())
	}
	return strings.TrimSpace(outBuff.String()), nil
}

func GitCloneCmd(url, targetDir, initMsg string) error {
	// Check if targetDir exists
	if _, err := os.Stat(targetDir); err == nil {
		return eris.Errorf("Game shard named '%s' already exists in this directory, "+
			"please change the directory or use another name", targetDir)
	}

	// Get latest tag from remote
	rev, err := git("ls-remote", "--tags", "--sort=-v:refname", url)
	if err != nil {
		return eris.Wrapf(err, "failed to fetch tags from remote")
	}

	// Extract latest tag
	lines := strings.Split(rev, "\n")
	if len(lines) == 0 {
		return eris.New("no tags found in the repository")
	}
	latestTag := strings.TrimPrefix(strings.Fields(lines[len(lines)-1])[1], "refs/tags/")

	// Clone only the latest tag with depth 1
	_, err = git("clone", "--branch", latestTag, "--depth", "1", url, targetDir)
	if err != nil {
		return eris.Wrapf(err, "failed to clone repository")
	}

	// Change directory to targetDir
	if err := os.Chdir(targetDir); err != nil {
		return eris.Wrapf(err, "failed to change directory")
	}

	// Remove the .git folder to reset commit history
	if err := os.RemoveAll(".git"); err != nil {
		return eris.Wrapf(err, "failed to remove .git folder")
	}

	// Reinitialize Git
	_, err = git("init")
	if err != nil {
		return eris.Wrapf(err, "failed to initialize git")
	}

	// Refactor module name
	if err := refactorModuleName(oldModuleName, filepath.Base(targetDir)); err != nil {
		return eris.Wrapf(err, "failed to refactor module name")
	}

	// Generate router key and update TOML
	rtrKey, err := generateRandomHexString(routerKeyLength)
	if err != nil {
		return eris.Wrapf(err, "failed to generate router key")
	}

	if err := appendToToml(tomlFile, tomlSectionCommon, map[string]any{routerKey: rtrKey}); err != nil {
		return eris.Wrapf(err, "failed to append to TOML")
	}

	// Commit the changes
	_, err = git("add", "-A")
	if err != nil {
		return eris.Wrapf(err, "failed to stage changes")
	}

	_, err = git("commit", "--author=\"World CLI <no-reply@world.dev>\"", "-m", initMsg)
	if err != nil {
		return eris.Wrapf(err, "failed to commit changes")
	}

	return nil
}

func refactorModuleName(oldModuleName, newModuleName string) error {
	cardinalDir := "cardinal"
	// Update import paths in all Go files
	err := filepath.Walk(cardinalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return eris.Wrapf(err, "failed to walk directory")
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			return replaceInFile(path, oldModuleName, newModuleName)
		}
		return nil
	})

	if err != nil {
		return eris.Wrapf(err, "failed to refactor module name")
	}

	// Update the go.mod file
	goModFilePath := filepath.Join(cardinalDir, "go.mod")

	return replaceInFile(goModFilePath, oldModuleName, newModuleName)
}

func replaceInFile(filePath, oldStr, newStr string) error {
	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return eris.Wrapf(err, "failed to open file")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, strings.ReplaceAll(scanner.Text(), oldStr, newStr))
	}
	if err := scanner.Err(); err != nil {
		return eris.Wrapf(err, "failed to scan file")
	}

	if err := file.Truncate(0); err != nil {
		return eris.Wrapf(err, "failed to truncate file")
	}
	if _, err := file.Seek(0, 0); err != nil {
		return eris.Wrapf(err, "failed to seek file")
	}

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return eris.Wrapf(err, "failed to write line")
		}
	}
	return writer.Flush()
}

// appendToToml reads the given TOML file, appends the specified section and fields, and writes it back.
func appendToToml(filePath, section string, fields map[string]any) error {
	// Read the existing content of the TOML file
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return eris.Wrap(err, "error reading file")
	}

	// Unmarshal the file content into a map
	var config map[string]any
	err = toml.Unmarshal(fileContent, &config)
	if err != nil {
		return eris.Wrap(err, "error unmarshaling TOML")
	}

	if config == nil {
		config = make(map[string]any)
	}

	// Check if the section already exists, if not, create it
	if _, exists := config[section]; !exists {
		config[section] = make(map[string]any)
	}

	// Add the fields to the section
	for key, value := range fields {
		config[section].(map[string]any)[key] = value
	}

	// Marshal the updated config back to TOML
	newContent, err := toml.Marshal(config)
	if err != nil {
		return eris.Wrap(err, "error marshaling TOML")
	}

	// Write the updated TOML back to the file
	err = os.WriteFile(filePath, newContent, 0600)
	if err != nil {
		return eris.Wrap(err, "error writing file")
	}
	return nil
}

func generateRandomHexString(length int) (string, error) {
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(randomBytes), nil
}
