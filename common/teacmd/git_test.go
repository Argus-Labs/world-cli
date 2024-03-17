package teacmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/magefile/mage/sh"
	"gotest.tools/v3/assert"
)

const templateURLTest = "https://github.com/Argus-Labs/starter-game-template.git"

func TestGitCloneCmd(t *testing.T) {
	type param struct {
		url       string
		targetDir string
		initMsg   string
	}

	test := []struct {
		name     string
		wantErr  bool
		expected int
		param    param
	}{
		{
			name:     "error clone wrong address",
			wantErr:  true,
			expected: 128,
			param: param{
				url:       "wrong address",
				targetDir: "targetDir",
				initMsg:   "initMsg",
			},
		},
		{
			name:    "success",
			wantErr: false,
			param: param{
				url:       templateURLTest,
				targetDir: filepath.Join(os.TempDir(), "worldclitest"),
				initMsg:   "initMsg",
			},
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			// clean up before test
			cleanUpDir(tt.param.targetDir)

			err := GitCloneCmd(tt.param.url, tt.param.targetDir, tt.param.initMsg)
			if tt.wantErr {
				assert.Equal(t, sh.ExitStatus(err), tt.expected)
			} else {
				assert.NilError(t, err)
			}

			// clean up after test
			cleanUpDir(tt.param.targetDir)
		})
	}
}

func cleanUpDir(targetDir string) {
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		err := os.RemoveAll(targetDir)
		if err != nil {
			fmt.Println(err)
		}
	}
}
