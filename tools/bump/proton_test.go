package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func protonFixture(version, extension string) protonRelease {
	release := protonRelease{CategoryName: "Stable", Version: version}
	release.File = append(release.File, struct {
		URL            string `json:"Url"`
		Sha512CheckSum string `json:"Sha512CheckSum"`
	}{URL: "https://example.test/pass-" + version + extension, Sha512CheckSum: strings.Repeat("ab", 64)})
	return release
}

func TestProtonLatestCommonStable(t *testing.T) {
	beta := protonFixture("2.0.0", ".deb")
	beta.CategoryName = "Beta"
	invalid := protonFixture("2.0.0", ".deb")
	invalid.File[0].Sha512CheckSum = "bad-checksum"

	for _, tt := range []struct {
		name         string
		linux, macos []protonRelease
		want         string
	}{
		{"Linux ahead", []protonRelease{protonFixture("1.43.1", ".deb"), protonFixture("1.43.0", ".deb")}, []protonRelease{protonFixture("1.43.0", ".dmg")}, "1.43.0"},
		{"macOS ahead", []protonRelease{protonFixture("1.43.0", ".deb")}, []protonRelease{protonFixture("1.43.1", ".dmg"), protonFixture("1.43.0", ".dmg")}, "1.43.0"},
		{"numeric order", []protonRelease{protonFixture("1.9.0", ".deb"), protonFixture("1.10.0", ".deb")}, []protonRelease{protonFixture("1.10.0", ".dmg"), protonFixture("1.9.0", ".dmg")}, "1.10.0"},
		{"no common version", []protonRelease{protonFixture("1.43.1", ".deb")}, []protonRelease{protonFixture("1.43.0", ".dmg")}, ""},
		{"missing artifact", []protonRelease{protonFixture("1.43.0", ".rpm")}, []protonRelease{protonFixture("1.43.0", ".dmg")}, ""},
		{"ignores beta", []protonRelease{beta, protonFixture("1.43.0", ".deb")}, []protonRelease{protonFixture("2.0.0", ".dmg"), protonFixture("1.43.0", ".dmg")}, "1.43.0"},
		{"requires valid checksum", []protonRelease{invalid, protonFixture("1.43.0", ".deb")}, []protonRelease{protonFixture("2.0.0", ".dmg"), protonFixture("1.43.0", ".dmg")}, "1.43.0"},
		{"empty feed", nil, []protonRelease{protonFixture("1.43.0", ".dmg")}, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				releases := tt.linux
				if r.URL.Path == "/macos" {
					releases = tt.macos
				}
				_ = json.NewEncoder(w).Encode(struct{ Releases []protonRelease }{releases})
			}))
			defer server.Close()
			feeds := map[string]string{"x86_64-linux": server.URL + "/linux", "aarch64-darwin": server.URL + "/macos"}
			got, err := protonLatestCommonStable(t.Context(), feeds)
			if tt.want == "" {
				require.ErrorContains(t, err, "no common Stable release")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			// A selected older release must remain fetchable when a platform is ahead.
			for system, url := range feeds {
				releases, err := fetchProtonReleases(t.Context(), url)
				require.NoError(t, err)
				artifact, err := protonArtifactAtVersion(releases, system, got)
				require.NoError(t, err)
				require.Contains(t, artifact.fields["url"], got)
			}
		})
	}
}
