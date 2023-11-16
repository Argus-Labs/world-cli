package config

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/pelletier/go-toml"
	"gotest.tools/v3/assert"
)

func getNamespace(t *testing.T, cfg Config) string {
	val, ok := cfg.Env["CARDINAL_NAMESPACE"]
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
	cfg, err := LoadConfig(file)
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
	cfg, err := LoadConfig("")
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}

func TestConfigPreference(t *testing.T) {
	fileConfig := makeConfigAtTemp(t, "alpha")
	envConfig := makeConfigAtTemp(t, "beta")
	replaceEnvVarForTest(t, WorldCLIConfigFileEnvVariable, envConfig)
	cfg, err := LoadConfig(fileConfig)
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

	cfg, err := LoadConfig("")
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

	cfg, err := LoadConfig("")
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}

func TestTextDecoding(t *testing.T) {
	content := `
[cardinal]
namespace="alpha"
`

	file, err := os.CreateTemp("", "config*.toml")
	assert.NilError(t, err)
	t.Cleanup(func() {
		assert.NilError(t, os.Remove(file.Name()))
	})
	_, err = fmt.Fprint(file, content)
	assert.NilError(t, err)
	file.Close()

	cfg, err := LoadConfig(file.Name())
	assert.NilError(t, err)
	assert.Equal(t, "alpha", getNamespace(t, cfg))
}
