package teacmd

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

type GitCloneProgressMsg struct {
	Status string
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
		return "", err
	}
	return outBuff.String(), nil
}

func GitCloneCmd(url string, targetDir string, initMsg string, progressChan chan<- GitCloneProgressMsg) error {
	defer func() {
		if progressChan != nil {
			close(progressChan)
		}
	}()

	// check targetDir exists
	if _, err := os.Stat(targetDir); err == nil {
		progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Error: Game shard named '%s' already exists", targetDir)}
		return eris.Errorf("Game shard named '%s' already exists in this directory, "+
			"please change the directory or use another name", targetDir)
	}

	progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Cloning %s into %s...", url, targetDir)}
	_, err := git("clone", url, targetDir)
	if err != nil {
		progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Error: Failed to clone repository: %v", err)}
		return err
	}
	progressChan <- GitCloneProgressMsg{Status: "Clone completed, checking out latest tag..."}

	err = os.Chdir(targetDir)
	if err != nil {
		progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Error: Failed to change directory: %v", err)}
		return err
	}

	rev, err := git("rev-list", "--tags", "--max-count=1")
	if err != nil {
		progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Error: Failed to get latest tag: %v", err)}
		return err
	}

	tag, err := git("describe", "--tags", strings.TrimSpace(rev))
	if err != nil {
		progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Error: Failed to describe tag: %v", err)}
		return err
	}

	_, err = git("checkout", strings.TrimSpace(tag))
	if err != nil {
		progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Error: Failed to checkout tag: %v", err)}
		return err
	}
	progressChan <- GitCloneProgressMsg{Status: "Latest tag checked out, initializing repository..."}

	err = os.RemoveAll(".git")
	if err != nil {
		progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Error: Failed to remove .git directory: %v", err)}
		return err
	}

	_, err = git("init")
	if err != nil {
		progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Error: Failed to initialize repository: %v", err)}
		return err
	}
	progressChan <- GitCloneProgressMsg{Status: "Repository initialized, updating module names..."}

	err = refactorModuleName(oldModuleName, filepath.Base(targetDir))
	if err != nil {
		progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Error: Failed to update module names: %v", err)}
		return err
	}

	rtrKey, err := generateRandomHexString(routerKeyLength)
	if err != nil {
		progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Error: Failed to generate router key: %v", err)}
		return err
	}

	err = appendToToml(tomlFile, tomlSectionCommon, map[string]any{routerKey: rtrKey})
	if err != nil {
		progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Error: Failed to update configuration: %v", err)}
		return err
	}
	progressChan <- GitCloneProgressMsg{Status: "Configuration updated, committing changes..."}

	_, err = git("add", "-A")
	if err != nil {
		progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Error: Failed to stage changes: %v", err)}
		return err
	}

	_, err = git("commit", "--author=\"World CLI <no-reply@world.dev>\"", "-m", initMsg)
	if err != nil {
		progressChan <- GitCloneProgressMsg{Status: fmt.Sprintf("Error: Failed to commit changes: %v", err)}
		return err
	}
	progressChan <- GitCloneProgressMsg{Status: "Repository setup completed successfully"}

	return nil
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
