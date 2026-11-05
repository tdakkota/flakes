package main

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtonArtifactForSystem(t *testing.T) {
	for _, tt := range []struct {
		name, system, version, url, checksum, wantURL, wantError string
	}{
		{name: "linux deb after rpm", system: "x86_64-linux", version: "1.43.0", url: "https://example.test/pass.deb", checksum: strings.Repeat("ab", 64), wantURL: "https://example.test/pass.deb"},
		{name: "macOS", system: "aarch64-darwin", version: "1.43.0", url: "https://example.test/pass.dmg", checksum: strings.Repeat("ab", 64), wantURL: "https://example.test/pass.dmg"},
		{name: "wrong version", system: "x86_64-linux", version: "1.42.0", url: "https://example.test/pass.deb", checksum: strings.Repeat("ab", 64), wantError: "requested version"},
		{name: "missing deb", system: "x86_64-linux", version: "1.43.0", url: "https://example.test/pass.zip", checksum: strings.Repeat("ab", 64), wantError: "no .deb artifact"},
		{name: "short checksum", system: "x86_64-linux", version: "1.43.0", url: "https://example.test/pass.deb", checksum: "ab", wantError: "checksum length"},
		{name: "invalid checksum", system: "x86_64-linux", version: "1.43.0", url: "https://example.test/pass.deb", checksum: strings.Repeat("zz", 64), wantError: "decode hex digest"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var release protonRelease
			release.Version = "1.43.0"
			file := struct {
				URL            string `json:"Url"`
				Sha512CheckSum string `json:"Sha512CheckSum"`
			}{}
			file.URL, file.Sha512CheckSum = "https://example.test/pass.rpm", strings.Repeat("cd", 64)
			release.File = append(release.File, file)
			file.URL, file.Sha512CheckSum = tt.url, tt.checksum
			release.File = append(release.File, file)
			got, err := protonArtifactForSystem(release, tt.system, tt.version)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantURL, got.fields["url"])
			require.Equal(t, "sha512-"+base64.StdEncoding.EncodeToString([]byte(strings.Repeat("\xab", 64))), got.sri)
		})
	}
}
