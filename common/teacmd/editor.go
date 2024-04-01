package teacmd

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"pkg.world.dev/world-cli/common/globalconfig"
)

const (
	TargetEditorDir = ".editor"

	latestReleaseURL             = "https://api.github.com/repos/Argus-Labs/cardinal-editor/releases/latest"
	httpTimeout                  = 2 * time.Second
	cardinalProjectIDPlaceholder = "__CARDINAL_PROJECT_ID__"
)

type Asset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Release struct {
	Name   string  `json:"name"`
	Assets []Asset `json:"assets"`
}

func SetupCardinalEditor() error {
	configDir, err := globalconfig.GetConfigDir()
	if err != nil {
		return err
	}

	editorDir, err := downloadReleaseIfNotCached(configDir)
	if err != nil {
		return err
	}

	// rename version tag dir to .editor
	err = copyDir(editorDir, TargetEditorDir)
	if err != nil {
		return err
	}

	// rename project id
	// "ce" prefix is added because guids can start with numbers, which is not allowed in js
	projectID := "ce" + strippedGUID()
	err = replaceProjectIDs(TargetEditorDir, projectID)
	if err != nil {
		return err
	}

	return nil
}

func downloadReleaseIfNotCached(configDir string) (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{
		Timeout: httpTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release Release
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err = json.Unmarshal(bodyBytes, &release); err != nil {
		return "", err
	}

	editorDir := filepath.Join(configDir, "editor")

	targetDir := filepath.Join(editorDir, release.Name)
	if _, err = os.Stat(targetDir); os.IsNotExist(err) {
		return targetDir, downloadAndUnzip(release.Assets[0].BrowserDownloadURL, targetDir)
	}

	return targetDir, nil
}

func downloadAndUnzip(url string, targetDir string) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: httpTimeout + 10*time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New(url)
	}
	defer resp.Body.Close()

	tmpZipFileName := "tmp.zip"
	file, err := os.Create(tmpZipFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	if err = unzipFile(tmpZipFileName, targetDir); err != nil {
		return err
	}

	return os.Remove(tmpZipFileName)
}

func unzipFile(filename string, targetDir string) error {
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return err
	}
	defer reader.Close()

	// save original folder name
	var originalDir string
	for i, file := range reader.File {
		if i == 0 {
			originalDir = file.Name
		}

		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		filePath, err := sanitizeExtractPath(filepath.Dir(targetDir), file.Name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			err = os.MkdirAll(filePath, 0755)
			if err != nil {
				return err
			}
			continue
		}

		dst, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer dst.Close()

		_, err = io.Copy(dst, src) //nolint:gosec // zip file is from us
		if err != nil {
			return err
		}
	}

	if err = os.Rename(filepath.Join(filepath.Dir(targetDir), originalDir), targetDir); err != nil {
		return err
	}

	return nil
}

func sanitizeExtractPath(dst string, filePath string) (string, error) {
	dstPath := filepath.Join(dst, filePath)
	if strings.HasPrefix(dstPath, filepath.Clean(dst)) {
		return dstPath, nil
	}
	return "", fmt.Errorf("%s: illegal file path", filePath)
}

func copyDir(src string, dst string) error {
	srcDir, err := os.ReadDir(src)
	if err != nil {
		return errors.New(src)
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, entry := range srcDir {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy dirs
			if err := copyDir(srcPath, destPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, destPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}

func replaceProjectIDs(editorDir string, newID string) error {
	assetsDir := filepath.Join(editorDir, "assets")
	files, err := os.ReadDir(assetsDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".js") {
			content, err := os.ReadFile(filepath.Join(assetsDir, file.Name()))
			if err != nil {
				return err
			}

			newContent := strings.ReplaceAll(string(content), cardinalProjectIDPlaceholder, newID)

			err = os.WriteFile(filepath.Join(assetsDir, file.Name()), []byte(newContent), 0600)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func strippedGUID() string {
	u := uuid.New()
	return strings.ReplaceAll(u.String(), "-", "")
}
