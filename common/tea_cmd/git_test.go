package tea_cmd

import (
	"fmt"
	"gotest.tools/v3/assert"
	"os"
	"testing"
)

const templateUrlTest = "https://github.com/Argus-Labs/starter-game-template.git"

func TestGitCloneCmd(t *testing.T) {
	type param struct {
		url       string
		targetDir string
		initMsg   string
	}

	test := []struct {
		name     string
		wantErr  bool
		expected string
		param    param
	}{
		{
			name:     "error clone wrong address",
			wantErr:  true,
			expected: `running "git clone wrong address targetDir" failed with exit code 128`,
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
				url:       templateUrlTest,
				targetDir: os.TempDir() + "/worldclitest",
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
				assert.Error(t, err, tt.expected)
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
