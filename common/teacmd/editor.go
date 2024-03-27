package teacmd

import (
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	latestReleaseURL             = "https://api.github.com/repos/Argus-Labs/cardinal-editor/releases/latest"
	httpTimeout                  = 5 * time.Second
	tmpZipFileName               = "tmp.zip"
	cardinalProjectIDPlaceholder = "__CARDINAL_PROJECT_ID__"
)

type Asset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Release struct {
	Assets []Asset `json:"assets"`
}

func SetupCardinalEditor() error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: httpTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var release Release
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(bodyBytes, &release); err != nil {
		return err
	}

	err = downloadRelease(release.Assets[0].BrowserDownloadURL)
	if err != nil {
		return err
	}

	if err = unzipFile(); err != nil {
		return err
	}

	return nil
}

func downloadRelease(url string) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: httpTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(tmpZipFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func unzipFile() error {
	reader, err := zip.OpenReader(tmpZipFileName)
	if err != nil {
		return err
	}
	defer reader.Close()

	// save original folder name
	var originalDir string
	for _, file := range reader.File {
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		if file.FileInfo().IsDir() {
			if originalDir == "" {
				originalDir = file.Name
			}
			err = os.MkdirAll(file.Name, 0777)
			if err != nil {
				return err
			}
			continue
		}

		dstPath := file.Name
		dst, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		_, err = io.Copy(dst, src) //nolint:gosec // zip file is from us
		if err != nil {
			return err
		}
	}

	if err = os.Rename(originalDir, ".editor"); err != nil {
		return err
	}

	if err = os.Remove(tmpZipFileName); err != nil {
		return err
	}

	if err = replaceProjectIDs(); err != nil {
		return err
	}

	return nil
}

func replaceProjectIDs() error {
	assetsDir := ".editor/assets"
	files, err := os.ReadDir(assetsDir)
	if err != nil {
		return err
	}

	// "ce" prefix is added because guids can start with numbers, which is not allowed in js
	projectID := "ce" + strippedGUID()
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".js") {
			content, err := os.ReadFile(filepath.Join(assetsDir, file.Name()))
			if err != nil {
				return err
			}

			newContent := strings.ReplaceAll(string(content), cardinalProjectIDPlaceholder, projectID)

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
