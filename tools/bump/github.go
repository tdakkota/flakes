package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-faster/errors"
)

type githubRelease struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name string `json:"name"`
	} `json:"assets"`
}

func githubJSON(ctx context.Context, path string, output any) error {
	url := "https://api.github.com/repos/" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrap(err, "build request")
	}
	req.Header.Set("User-Agent", "flakes-bump/1")
	req.Header.Set("Accept", "application/vnd.github+json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "GET %s", url)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.Errorf("GET %s: unexpected status %s", url, response.Status)
	}
	if err := json.NewDecoder(response.Body).Decode(output); err != nil {
		return errors.Wrap(err, "decode GitHub response")
	}
	return nil
}

func githubLatestTag(ctx context.Context, repo, trimPrefix string) (string, error) {
	var release githubRelease
	if err := githubJSON(ctx, repo+"/releases/latest", &release); err != nil {
		return "", err
	}
	return strings.TrimPrefix(release.TagName, trimPrefix), nil
}

func githubLatestTagWithAssets(ctx context.Context, repo, trimPrefix, prefix string) (string, error) {
	var releases []githubRelease
	if err := githubJSON(ctx, repo+"/releases?per_page=100", &releases); err != nil {
		return "", err
	}
	return latestReleaseWithAssets(releases, trimPrefix, prefix)
}

func latestReleaseWithAssets(releases []githubRelease, trimPrefix, prefix string) (string, error) {
	var latest string
	for _, release := range releases {
		if release.Draft || release.Prerelease || !strings.HasPrefix(release.TagName, trimPrefix) {
			continue
		}
		version := strings.TrimPrefix(release.TagName, trimPrefix)
		assets := make(map[string]bool, len(release.Assets))
		for _, asset := range release.Assets {
			assets[asset.Name] = true
		}
		compatible := true
		for _, platform := range []string{"darwin-aarch64", "linux-aarch64", "linux-x86_64"} {
			if !assets[fmt.Sprintf("%s-%s-%s.zip", prefix, platform, version)] {
				compatible = false
				break
			}
		}
		if compatible && (latest == "" || compareVersions(version, latest) > 0) {
			latest = version
		}
	}
	if latest == "" {
		return "", errors.Errorf("no stable release publishes all %s artifacts", prefix)
	}
	return latest, nil
}
