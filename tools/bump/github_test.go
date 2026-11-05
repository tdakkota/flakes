package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func releaseFixture(version, prefix string) githubRelease {
	release := githubRelease{TagName: "v" + version}
	for _, platform := range []string{"darwin-aarch64", "linux-aarch64", "linux-x86_64"} {
		release.Assets = append(release.Assets, struct {
			Name string `json:"name"`
		}{fmt.Sprintf("%s-%s-%s.zip", prefix, platform, version)})
	}
	return release
}

func TestLatestReleaseWithAssets(t *testing.T) {
	draft := releaseFixture("4.0.0", "vibe")
	draft.Draft = true
	prerelease := releaseFixture("3.2.0", "vibe")
	prerelease.Prerelease = true
	unprefixed := releaseFixture("3.0.0", "vibe")
	unprefixed.TagName = "3.0.0"
	for _, tt := range []struct {
		name, prefix, want string
		releases           []githubRelease
	}{
		{"CLI skips ACP-only release", "vibe", "2.24.5", []githubRelease{releaseFixture("3.1.0", "vibe-acp"), releaseFixture("2.24.5", "vibe")}},
		{"ACP updates independently", "vibe-acp", "3.1.0", []githubRelease{releaseFixture("2.24.5", "vibe"), releaseFixture("3.1.0", "vibe-acp")}},
		{"ignores drafts and prereleases", "vibe", "2.24.5", []githubRelease{draft, prerelease, releaseFixture("2.24.5", "vibe")}},
		{"requires every platform", "vibe", "", []githubRelease{{TagName: "v3.1.0"}}},
		{"requires the download tag prefix", "vibe", "", []githubRelease{unprefixed}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := latestReleaseWithAssets(tt.releases, "v", tt.prefix)
			if tt.want == "" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func FuzzLatestReleaseWithAssets(f *testing.F) {
	for _, seed := range [][]githubRelease{
		nil,
		{releaseFixture("2.24.5", "vibe")},
		{releaseFixture("3.1.0", "vibe-acp"), releaseFixture("2.24.5", "vibe")},
	} {
		data, err := json.Marshal(seed)
		require.NoError(f, err)
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var releases []githubRelease
		if json.Unmarshal(data, &releases) != nil {
			return
		}
		version, err := latestReleaseWithAssets(releases, "v", "vibe")
		if err != nil {
			return
		}
		for _, release := range releases {
			if release.TagName != "v"+version || release.Draft || release.Prerelease {
				continue
			}
			var names []string
			for _, asset := range release.Assets {
				names = append(names, asset.Name)
			}
			if len(names) < 3 {
				continue
			}
			valid := true
			for _, platform := range []string{"darwin-aarch64", "linux-aarch64", "linux-x86_64"} {
				valid = valid && slices.Contains(names, "vibe-"+platform+"-"+version+".zip")
			}
			if valid {
				return
			}
		}
		t.Fatal("selected a release without compatible stable assets")
	})
}
