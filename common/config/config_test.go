package config

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

// cmdZero returns an empty cobra command. Since the config flag is not set, GetConfig will search
// the local directory (and parent directories) for a config file.
func cmdZero() *cobra.Command {
	return &cobra.Command{}
}

// cmdWithConfig creates a command that has the --config flag set to the given filename
func cmdWithConfig(filename string) *cobra.Command {
	cmd := cmdZero()
	AddConfigFlag(cmd)
	cmd.Flags().Set(flagForConfigFile, filename)
	return cmd
}

func getNamespace(t *testing.T, cfg Config) string {
	val, ok := cfg.DockerEnv["CARDINAL_NAMESPACE"]
	assert.Check(t, ok, "no CARDINAL_NAMESPACE field found")
	return val
}

func makeConfigAtTemp(t *testing.T, namespace string) (filename string) {
	file, err := os.CreateTemp("", "config*.toml")
	assert.NilError(t, err)
	defer file.Close()
	t.Cleanup(func() {
		assert.NilError(t, os.Remove(file.Name()))
	})
	makeConfigAtFile(t, file, namespace)
	return file.Name()
}

func makeConfigAtPath(t *testing.T, path, namespace string) {
	file, err := os.Create(path)
	assert.NilError(t, err)
	defer file.Close()
	makeConfigAtFile(t, file, namespace)
}

func makeConfigAtFile(t *testing.T, file *os.File, namespace string) {
	data := map[string]any{
		"cardinal": map[string]any{
			"CARDINAL_NAMESPACE": namespace,
		},
	}
	assert.NilError(t, toml.NewEncoder(file).Encode(data))
}

func TestCanSetNamespaceWithFilename(t *testing.T) {
	file := makeConfigAtTemp(t, "alpha")
	cfg, err := GetConfig(cmdWithConfig(file))
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}

func replaceEnvVarForTest(t *testing.T, env, value string) {
	original := os.Getenv(env)
	t.Cleanup(func() {
		assert.NilError(t, os.Setenv(env, original))
	})
	os.Setenv(env, value)
}

func TestCanSetNamespaceWithEnvVariable(t *testing.T) {
	file := makeConfigAtTemp(t, "alpha")
	replaceEnvVarForTest(t, WorldCLIConfigFileEnvVariable, file)
	cfg, err := GetConfig(cmdZero())
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}

func TestConfigPreference(t *testing.T) {
	fileConfig := makeConfigAtTemp(t, "alpha")
	envConfig := makeConfigAtTemp(t, "beta")
	replaceEnvVarForTest(t, WorldCLIConfigFileEnvVariable, envConfig)
	cfg, err := GetConfig(cmdWithConfig(fileConfig))
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}

func makeTempDir(t *testing.T) string {
	tempdir, err := os.MkdirTemp("", "")
	assert.NilError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tempdir)
	})

	// cd over to the temporary directory. Make sure to jump back to the current directory
	// at the end of the test
	currDir, err := os.Getwd()
	assert.NilError(t, err)
	t.Cleanup(func() {
		os.Chdir(currDir)
	})
	assert.NilError(t, os.Chdir(tempdir))
	return tempdir
}

func TestConfigFromLocalFile(t *testing.T) {
	tempdir := makeTempDir(t)

	configFile := path.Join(tempdir, WorldCLIConfigFilename)
	makeConfigAtPath(t, configFile, "alpha")

	cfg, err := GetConfig(cmdZero())
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}

func TestLoadConfigLooksInParentDirectories(t *testing.T) {
	tempdir := makeTempDir(t)

	deepPath := path.Join(tempdir, "/a/b/c/d/e/f/g")
	assert.NilError(t, os.MkdirAll(deepPath, 0755))

	configFile := path.Join(deepPath, WorldCLIConfigFilename)
	// The eventual call to LoadConfig should find this config file
	makeConfigAtPath(t, configFile, "alpha")
	deepPath = path.Join(deepPath, "/h/i/j/k/l/m/n")
	assert.NilError(t, os.MkdirAll(deepPath, 0755))
	assert.NilError(t, os.Chdir(deepPath))

	cfg, err := GetConfig(cmdZero())
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}

func makeTempConfigWithContent(t *testing.T, content string) (filename string) {
	file, err := os.CreateTemp("", "config*.toml")
	assert.NilError(t, err)
	defer file.Close()
	t.Cleanup(func() {
		assert.NilError(t, os.Remove(file.Name()))
	})
	_, err = fmt.Fprint(file, content)
	assert.NilError(t, err)
	return file.Name()
}

func TestTextDecoding(t *testing.T) {
	content := `
[cardinal]
CARDINAL_NAMESPACE="alpha"
`
	filename := makeTempConfigWithContent(t, content)

	cfg, err := GetConfig(cmdWithConfig(filename))
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}

func TestCanSetArbitraryEnvVariables(t *testing.T) {
	content := `
[evm]
ENV_ALPHA="alpha"

[cardinal]
ENV_BETA="beta"
`
	filename := makeTempConfigWithContent(t, content)

	cfg, err := GetConfig(cmdWithConfig(filename))
	assert.NilError(t, err)
	assert.Equal(t, cfg.DockerEnv["ENV_ALPHA"], "alpha")
	assert.Equal(t, cfg.DockerEnv["ENV_BETA"], "beta")
}

func TestCanOverrideRootDir(t *testing.T) {
	content := `
[evm]
FOO = "bar"
`
	filename := makeTempConfigWithContent(t, content)
	// by default, the root path should match the location of the toml file.
	wantRootDir, _ := path.Split(filename)
	cfg, err := GetConfig(cmdWithConfig(filename))
	assert.NilError(t, err)
	assert.Equal(t, wantRootDir, cfg.RootDir)
	assert.Equal(t, cfg.DockerEnv["FOO"], "bar")

	// Alternatively, a custom root dir can be set in the congif file
	content = `
root_dir="/some/crazy/path"
[cardinal]
FOO = "bar"
`
	wantRootDir = "/some/crazy/path"
	filename = makeTempConfigWithContent(t, content)
	cfg, err = GetConfig(cmdWithConfig(filename))
	assert.NilError(t, err)
	assert.Equal(t, wantRootDir, cfg.RootDir)
	assert.Equal(t, "bar", cfg.DockerEnv["FOO"])
}

func TestErrorWhenNoConfigFileExists(t *testing.T) {
	_, err := GetConfig(cmdZero())
	assert.Check(t, err != nil)
}

func TestNumbersAreValidDockerEnvVariable(t *testing.T) {
	content := `
[evm]
SOME_INT = 100
SOME_FLOAT = 99.9
`
	filename := makeTempConfigWithContent(t, content)
	cfg, err := GetConfig(cmdWithConfig(filename))
	assert.NilError(t, err)
	assert.Equal(t, "100", cfg.DockerEnv["SOME_INT"])
	assert.Equal(t, "99.9", cfg.DockerEnv["SOME_FLOAT"])
}

func TestErrorOnInvalidToml(t *testing.T) {
	invalidContent := `
[cardinal]
SOME_INT = 100
SOME_FLOAT = 99.9
=1000
`
	filename := makeTempConfigWithContent(t, invalidContent)
	_, err := GetConfig(cmdWithConfig(filename))
	assert.Check(t, err != nil)
}

func TestDuplicateEnvironmentVariableProducesError(t *testing.T) {
	testCases := []struct {
		name string
		toml string
	}{
		{
			name: "malformed toml",
			toml: `
[cardinal]
SOME_INT = 100
SOME_FLOAT = 99.9
=1000
`,
		},
		{
			name: "duplicate env var in section",
			toml: `
[cardinal]
DUPLICATE = 100
DUPLICATE = 200
`,
		},
		{
			name: "duplicate env var in two section",
			toml: `
[evm]
DUPLICATE = 100
[cardinal]
DUPLICATE = 200
`,
		},
	}

	for _, tc := range testCases {
		filename := makeTempConfigWithContent(t, tc.toml)
		_, err := GetConfig(cmdWithConfig(filename))
		assert.Check(t, err != nil, "in %q", tc.name)
	}
}

func TestCanParseExampleConfig(t *testing.T) {
	exampleConfig := "../../example-world.toml"
	cfg, err := GetConfig(cmdWithConfig(exampleConfig))
	assert.NilError(t, err)
	assert.Equal(t, "my-world-1", cfg.DockerEnv["CARDINAL_NAMESPACE"])
	assert.Equal(t, "world-engine", cfg.DockerEnv["CHAIN_ID"])
}

func TestConfigFlagCannotBeEmpty(t *testing.T) {
	// If you set the config file, it cannot be empty
	_, err := GetConfig(cmdWithConfig(""))
	assert.Check(t, err != nil)
}
