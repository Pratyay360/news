package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
)

type Release struct {
	Assets []Asset `json:"assets"`
}

type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

func makko() {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	fmt.Printf("Detected OS: %s, Arch: %s\n", goos, goarch)

	url := "https://forge.starlightnet.work/api/v1/repos/Team/makko/releases/latest"
	resp, err := http.Get(url)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch release info")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("API returned non-OK status: %d", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read response body")
		return
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		log.Error().Err(err).Msg("Failed to decode JSON")
		return
	}

	var matchedAsset *Asset
	for _, asset := range release.Assets {
		name := asset.Name
		if strings.Contains(name, goos) && strings.Contains(name, goarch) {
			matchedAsset = &asset
			break
		}
	}
	// fallback: match OS only (backwards compat)
	if matchedAsset == nil {
		for _, asset := range release.Assets {
			if strings.Contains(asset.Name, goos) {
				matchedAsset = &asset
				break
			}
		}
	}

	if matchedAsset == nil {
		log.Warn().Msgf("No asset found matching %s/%s", goos, goarch)
		return
	}

	if err := downloadFile(matchedAsset.DownloadURL, matchedAsset.Name); err != nil {
		log.Error().Err(err).Msg("Download failed")
		return
	}

	fmt.Printf("Successfully downloaded %s\n", matchedAsset.Name)
}

func downloadFile(url, filename string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	out, err := os.Create(filename)
	os.Chmod(filename, 0755)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil

}

func main() {
	makko()
}
