package teacmd

import (
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	mock_terminal "pkg.world.dev/world-cli/utils/terminal/mock"
)

func Test_GitCloneCmd(t *testing.T) {
	terminalMock := mock_terminal.NewMockTerminal(gomock.NewController(t))
	cmd := New(terminalMock)

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
		mock     func()
	}{
		{
			name:     "error clone wrong address",
			wantErr:  true,
			expected: `exit status 128`,
			param: param{
				url:       "testUrl",
				targetDir: "targetDir",
				initMsg:   "initMsg",
			},
			mock: func() {
				terminalMock.EXPECT().Exec("git", "clone", "testUrl", "targetDir").
					Return([]byte(""), fmt.Errorf(`exit status 128`))
			},
		},
		{
			name:     "error change dir",
			wantErr:  true,
			expected: `error change dir`,
			param: param{
				url:       "testUrl",
				targetDir: "targetDir",
				initMsg:   "initMsg",
			},
			mock: func() {
				terminalMock.EXPECT().Exec("git", "clone", "testUrl", "targetDir").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Chdir("targetDir").Return(fmt.Errorf(`error change dir`))
			},
		},
		{
			name:     "error rev-list",
			wantErr:  true,
			expected: `error rev-list`,
			param: param{
				url:       "testUrl",
				targetDir: "targetDir",
				initMsg:   "initMsg",
			},
			mock: func() {
				terminalMock.EXPECT().Exec("git", "clone", "testUrl", "targetDir").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Chdir("targetDir").Return(nil)
				terminalMock.EXPECT().Exec("git", "rev-list", "--tags", "--max-count=1").
					Return([]byte(""), fmt.Errorf(`error rev-list`))
			},
		},
		{
			name:     "error describe",
			wantErr:  true,
			expected: `error describe`,
			param: param{
				url:       "testUrl",
				targetDir: "targetDir",
				initMsg:   "initMsg",
			},
			mock: func() {
				terminalMock.EXPECT().Exec("git", "clone", "testUrl", "targetDir").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Chdir("targetDir").Return(nil)
				terminalMock.EXPECT().Exec("git", "rev-list", "--tags", "--max-count=1").
					Return([]byte("rev"), nil)
				terminalMock.EXPECT().Exec("git", "describe", "--tags", "rev").
					Return([]byte(""), fmt.Errorf(`error describe`))
			},
		},
		{
			name:     "error checkout",
			wantErr:  true,
			expected: `error checkout`,
			param: param{
				url:       "testUrl",
				targetDir: "targetDir",
				initMsg:   "initMsg",
			},
			mock: func() {
				terminalMock.EXPECT().Exec("git", "clone", "testUrl", "targetDir").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Chdir("targetDir").Return(nil)
				terminalMock.EXPECT().Exec("git", "rev-list", "--tags", "--max-count=1").
					Return([]byte("rev"), nil)
				terminalMock.EXPECT().Exec("git", "describe", "--tags", "rev").
					Return([]byte("tag"), nil)
				terminalMock.EXPECT().Exec("git", "checkout", "tag").
					Return([]byte(""), fmt.Errorf(`error checkout`))
			},
		},
		{
			name:     "error rm",
			wantErr:  true,
			expected: `error rm`,
			param: param{
				url:       "testUrl",
				targetDir: "targetDir",
				initMsg:   "initMsg",
			},
			mock: func() {
				terminalMock.EXPECT().Exec("git", "clone", "testUrl", "targetDir").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Chdir("targetDir").Return(nil)
				terminalMock.EXPECT().Exec("git", "rev-list", "--tags", "--max-count=1").
					Return([]byte("rev"), nil)
				terminalMock.EXPECT().Exec("git", "describe", "--tags", "rev").
					Return([]byte("tag"), nil)
				terminalMock.EXPECT().Exec("git", "checkout", "tag").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Rm(".git").Return(fmt.Errorf(`error rm`))
			},
		},
		{
			name:     "error init",
			wantErr:  true,
			expected: `error init`,
			param: param{
				url:       "testUrl",
				targetDir: "targetDir",
				initMsg:   "initMsg",
			},
			mock: func() {
				terminalMock.EXPECT().Exec("git", "clone", "testUrl", "targetDir").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Chdir("targetDir").Return(nil)
				terminalMock.EXPECT().Exec("git", "rev-list", "--tags", "--max-count=1").
					Return([]byte("rev"), nil)
				terminalMock.EXPECT().Exec("git", "describe", "--tags", "rev").
					Return([]byte("tag"), nil)
				terminalMock.EXPECT().Exec("git", "checkout", "tag").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Rm(".git").Return(nil)
				terminalMock.EXPECT().Exec("git", "init").
					Return([]byte(""), fmt.Errorf(`error init`))
			},
		},
		{
			name:     "error add",
			wantErr:  true,
			expected: `error add`,
			param: param{
				url:       "testUrl",
				targetDir: "targetDir",
				initMsg:   "initMsg",
			},
			mock: func() {
				terminalMock.EXPECT().Exec("git", "clone", "testUrl", "targetDir").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Chdir("targetDir").Return(nil)
				terminalMock.EXPECT().Exec("git", "rev-list", "--tags", "--max-count=1").
					Return([]byte("rev"), nil)
				terminalMock.EXPECT().Exec("git", "describe", "--tags", "rev").
					Return([]byte("tag"), nil)
				terminalMock.EXPECT().Exec("git", "checkout", "tag").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Rm(".git").Return(nil)
				terminalMock.EXPECT().Exec("git", "init").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Exec("git", "add", "-A").
					Return([]byte(""), fmt.Errorf(`error add`))
			},
		},
		{
			name:     "error commit",
			wantErr:  true,
			expected: `error commit`,
			param: param{
				url:       "testUrl",
				targetDir: "targetDir",
				initMsg:   "initMsg",
			},
			mock: func() {
				terminalMock.EXPECT().Exec("git", "clone", "testUrl", "targetDir").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Chdir("targetDir").Return(nil)
				terminalMock.EXPECT().Exec("git", "rev-list", "--tags", "--max-count=1").
					Return([]byte("rev"), nil)
				terminalMock.EXPECT().Exec("git", "describe", "--tags", "rev").
					Return([]byte("tag"), nil)
				terminalMock.EXPECT().Exec("git", "checkout", "tag").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Rm(".git").Return(nil)
				terminalMock.EXPECT().Exec("git", "init").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Exec("git", "add", "-A").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Exec("git", "commit", "-m", "initMsg").
					Return([]byte(""), fmt.Errorf(`error commit`))
			},
		},
		{
			name:     "success",
			wantErr:  false,
			expected: `success`,
			param: param{
				url:       "testUrl",
				targetDir: "targetDir",
				initMsg:   "initMsg",
			},
			mock: func() {
				terminalMock.EXPECT().Exec("git", "clone", "testUrl", "targetDir").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Chdir("targetDir").Return(nil)
				terminalMock.EXPECT().Exec("git", "rev-list", "--tags", "--max-count=1").
					Return([]byte("rev"), nil)
				terminalMock.EXPECT().Exec("git", "describe", "--tags", "rev").
					Return([]byte("tag"), nil)
				terminalMock.EXPECT().Exec("git", "checkout", "tag").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Rm(".git").Return(nil)
				terminalMock.EXPECT().Exec("git", "init").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Exec("git", "add", "-A").
					Return([]byte(""), nil)
				terminalMock.EXPECT().Exec("git", "commit", "-m", "initMsg").
					Return([]byte(""), nil)
			},
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			// exec mock
			tt.mock()

			// exec function
			err := cmd.GitCloneCmd(tt.param.url, tt.param.targetDir, tt.param.initMsg)
			if tt.wantErr {
				assert.Error(t, err, tt.expected)
			} else {
				assert.NilError(t, err)
			}
		})
	}
}
