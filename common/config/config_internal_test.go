package config

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/pelletier/go-toml"
	"gotest.tools/v3/assert"
)

func getNamespace(t *testing.T, cfg *Config) string {
	val, ok := cfg.DockerEnv["CARDINAL_NAMESPACE"]
	assert.Check(t, ok, "no CARDINAL_NAMESPACE field found")
	return val
}

func makeConfigAtTemp(t *testing.T, namespace string) string {
	file, err := os.CreateTemp(t.TempDir(), "config*.toml")
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

	cfg, err := GetConfig(&file)
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}

func replaceEnvVarForTest(t *testing.T, env, value string) {
	t.Setenv(env, value)
}

func TestCanSetNamespaceWithEnvVariable(t *testing.T) {
	file := makeConfigAtTemp(t, "alpha")
	replaceEnvVarForTest(t, WorldCLIConfigFileEnvVariable, file)
	cfg, err := GetConfig(nil)
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}

func TestConfigPreference(t *testing.T) {
	fileConfig := makeConfigAtTemp(t, "alpha")
	envConfig := makeConfigAtTemp(t, "beta")
	replaceEnvVarForTest(t, WorldCLIConfigFileEnvVariable, envConfig)

	cfg, err := GetConfig(&fileConfig)
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}

func makeTempDir(t *testing.T) string {
	tempdir := t.TempDir()

	// Save current dir and restore later
	currDir, err := os.Getwd()
	assert.NilError(t, err)
	t.Cleanup(func() {
		t.Chdir(currDir)
	})

	t.Chdir(tempdir)
	return tempdir
}

func TestConfigFromLocalFile(t *testing.T) {
	tempdir := makeTempDir(t)

	configPath := path.Join(tempdir, WorldCLIConfigFilename)
	makeConfigAtPath(t, configPath, "alpha")

	cfg, err := GetConfig(nil)
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}

func TestLoadConfigLooksInParentDirectories(t *testing.T) {
	tempdir := makeTempDir(t)

	deepPath := path.Join(tempdir, "/a/b/c/d/e/f/g")
	assert.NilError(t, os.MkdirAll(deepPath, 0755))

	deepPath = path.Join(deepPath, "/h/i/j/k/l/m/n")
	assert.NilError(t, os.MkdirAll(deepPath, 0755))

	t.Chdir(deepPath)

	configFilePath := path.Join(deepPath, WorldCLIConfigFilename)
	// The eventual call to LoadConfig should find this config file
	makeConfigAtPath(t, configFilePath, "alpha")

	cfg, err := GetConfig(nil)
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}

func makeTempConfigWithContent(t *testing.T, content string) string {
	file, err := os.CreateTemp(t.TempDir(), "config*.toml")
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

	cfg, err := GetConfig(&filename)
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

	cfg, err := GetConfig(&filename)
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
	cfg, err := GetConfig(&filename)
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
	cfg, err = GetConfig(&filename)
	assert.NilError(t, err)
	assert.Equal(t, wantRootDir, cfg.RootDir)
	assert.Equal(t, "bar", cfg.DockerEnv["FOO"])
}

func TestErrorWhenNoConfigFileExists(t *testing.T) {
	_, err := GetConfig(nil)
	assert.Check(t, err != nil)
}

func TestNumbersAreValidDockerEnvVariable(t *testing.T) {
	content := `
[evm]
SOME_INT = 100
SOME_FLOAT = 99.9
`
	filename := makeTempConfigWithContent(t, content)
	cfg, err := GetConfig(&filename)
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
	_, err := GetConfig(&filename)
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
		_, err := GetConfig(&filename)
		assert.Check(t, err != nil, "in %q", tc.name)
	}
}

func TestCanParseExampleConfig(t *testing.T) {
	exampleConfig := "../../example-world.toml"
	cfg, err := GetConfig(&exampleConfig)
	assert.NilError(t, err)
	assert.Equal(t, "my-world-1", cfg.DockerEnv["CARDINAL_NAMESPACE"])
	assert.Equal(t, "world-engine", cfg.DockerEnv["CHAIN_ID"])
}
