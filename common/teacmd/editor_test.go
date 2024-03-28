package teacmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestSetupCardinalEditor(t *testing.T) {
	t.Run("setup cardinal editor", func(t *testing.T) {
		cleanUpDir(editorDir)

		err := SetupCardinalEditor()
		assert.NilError(t, err)

		// check if .editor directory exists
		_, err = os.Stat(editorDir)
		assert.NilError(t, err)

		// check if it's not empty
		dir, err := os.ReadDir(editorDir)
		assert.NilError(t, err)
		assert.Assert(t, len(dir) != 0)

		// check if project id is replaced
		containsPlaceholder, err := containsCardinalProjectIDPlaceholder("")
		assert.NilError(t, err)
		assert.Equal(t, containsPlaceholder, false)

		// TODO: check if cardinal editor works

		cleanUpDir(editorDir)
	})
}

func containsCardinalProjectIDPlaceholder(dir string) (bool, error) {
	files, err := os.ReadDir(filepath.Join(editorDir, dir))
	if err != nil {
		return false, err
	}

	for _, file := range files {
		// recurse over child directories
		if file.IsDir() {
			contains, err := containsCardinalProjectIDPlaceholder(filepath.Join(dir, file.Name()))
			if contains || err != nil {
				return contains, err
			}
			continue
		}

		if strings.HasSuffix(file.Name(), ".js") {
			filePath := filepath.Join(editorDir, dir, file.Name())

			content, err := os.ReadFile(filePath)
			if err != nil {
				return false, err
			}

			if strings.Contains(string(content), cardinalProjectIDPlaceholder) {
				return true, nil
			}
		}
	}

	return false, nil
}
