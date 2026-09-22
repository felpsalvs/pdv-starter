// Package update implements the PDV binary's self-update: check the
// GitHub Releases API for a newer tagged release, verify the matching
// platform archive's checksum, and swap it in for the running executable.
// It's a small purpose-built client for a single public GitHub repo,
// rather than a general multi-forge updater library — this app only ever
// updates itself from github.com/felpsalvs/pdv-starter releases.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	Owner = "felpsalvs"
	Repo  = "pdv-starter"

	// apiTimeout bounds the "check for updates" call on startup — offline
	// or a slow network must never stop the PDV from opening.
	apiTimeout = 8 * time.Second
)

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

func (r *Release) Asset(name string) (Asset, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

// FetchLatest returns the latest published (non-draft, non-prerelease)
// GitHub release for this repo.
func FetchLatest(ctx context.Context) (*Release, error) {
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", Owner, Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases: status %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func downloadAll(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: status %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
