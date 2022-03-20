package helm

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const (
	// InstallationDirectory is temporary folder name for caching downloaded installation packages.
	InstallationDirectory = "kubeedge-install-packages"
)

// URLFetcher is used to fetch and manipulate charts from remote url
type URLFetcher struct {
	url     string
	destDir string
	subDir  string
}

// NewURLFetcher creates a url fetcher for the online profile.
func NewURLFetcher(url, destDir, subDir string) URLFetcher {
	if destDir == "" {
		destDir = filepath.Join(os.TempDir(), InstallationDirectory)
	}
	return URLFetcher{
		url:     url,
		destDir: destDir,
		subDir:  subDir,
	}
}

// DownloadTo downloads from remote srcURL to dest local file path
func (f URLFetcher) DownloadTo() (string, error) {
	u, err := url.Parse(f.url)
	if err != nil {
		return "", fmt.Errorf("invalid chart URL: %s", f.url)
	}

	name := filepath.Base(u.Path)
	destFile := filepath.Join(f.destDir, f.subDir, name)
	if _, err = os.Stat(destFile); err == nil {
		return destFile, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("file stats error: %s", err.Error())
	}

	client := http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Get(f.url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch URL %s : %s", f.url, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(destFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, 0o755)
		if err != nil {
			return "", err
		}
	}

	if err := os.WriteFile(destFile, data, 0o644); err != nil {
		return destFile, err
	}

	return destFile, nil
}
