package teacmd

import (
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	latestReleaseURL = "https://api.github.com/repos/Argus-Labs/cardinal-editor/releases/latest"
	httpTimeout      = 2 * time.Second
)

type Asset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Release struct {
	Assets []Asset `json:"assets"`
}

func SetupCardinalEditor(targetDir string) error {
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

	err = downloadAndUnzipFle(release.Assets[0].BrowserDownloadURL, targetDir)
	if err != nil {
		return err
	}

	return nil
}

func downloadAndUnzipFle(url string, targetDir string) error {
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

	err = os.Chdir(targetDir)
	if err != nil {
		return err
	}

	err = unzipFile(resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func unzipFile(src io.Reader) error {
	tmp := "tmp.zip"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, src)
	if err != nil {
		return err
	}

	reader, err := zip.OpenReader(tmp)
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

	err = os.Rename(originalDir, ".editor")
	if err != nil {
		println(err.Error())
		return err
	}

	err = os.Remove(tmp)
	if err != nil {
		println(err.Error())
		return err
	}

	return nil
}
