package teacmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

const (
	testDir       = ".test-worldcli"
	testTargetDir = ".test-worldcli/.editor"
)

func TestSetupCardinalEditor(t *testing.T) {
	t.Run("setup cardinal editor", func(t *testing.T) {
		cleanUpDir(testDir)

		editorDir, err := downloadReleaseIfNotCached(testDir)
		assert.NilError(t, err)

		// check if editor directory exists
		_, err = os.Stat(editorDir)
		exists := os.IsNotExist(err)
		assert.Equal(t, exists, false)

		// check if it's not empty
		dir, err := os.ReadDir(editorDir)
		assert.NilError(t, err)
		assert.Assert(t, len(dir) != 0)

		// check if folder is renamed
		err = copyDir(editorDir, testTargetDir)
		assert.NilError(t, err)

		_, err = os.Stat(testTargetDir)
		exists = os.IsNotExist(err)
		assert.Equal(t, exists, false)

		// check if project id is replaced
		projectID := "__THIS_IS_FOR_TESTING_ONLY__"
		err = replaceProjectIDs(testTargetDir, projectID)
		assert.NilError(t, err)

		containsNewID, err := containsCardinalProjectIDPlaceholder(testTargetDir, projectID)
		assert.NilError(t, err)
		assert.Equal(t, containsNewID, true)

		// TODO: check if cardinal editor works

		cleanUpDir(testDir)
	})
}

func containsCardinalProjectIDPlaceholder(dir string, originalID string) (bool, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	for _, file := range files {
		// recurse over child directories
		if file.IsDir() {
			contains, err := containsCardinalProjectIDPlaceholder(filepath.Join(dir, file.Name()), originalID)
			if contains || err != nil {
				return contains, err
			}
			continue
		}

		if strings.HasSuffix(file.Name(), ".js") {
			filePath := filepath.Join(dir, file.Name())

			content, err := os.ReadFile(filePath)
			if err != nil {
				return false, err
			}

			if strings.Contains(string(content), originalID) {
				return true, nil
			}
		}
	}

	return false, nil
}
