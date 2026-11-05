package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyBump(t *testing.T) {
	for _, updateURL := range []bool{false, true} {
		t.Run(fmt.Sprintf("update_url_%t", updateURL), func(t *testing.T) {
			t.Parallel()
			v := versions{"app": {
				Version: "1.0",
				Artifacts: map[string]map[string]string{
					"x86_64-linux": {"hash": "old-hash", "url": "https://example.test/1.0.deb", "arch": "amd64"},
				},
			}}
			group := versionGroup{targets: []target{{nixKey: "app", fetch: func(_ context.Context, system string, fields map[string]string, version string) (artifact, error) {
				require.Equal(t, "x86_64-linux", system)
				require.Equal(t, "1.1", version)
				result := artifact{sri: "new-hash"}
				if updateURL {
					result.fields = map[string]string{"url": "https://example.test/1.1.deb"}
				}
				return result, nil
			}}}}
			require.NoError(t, applyBump(t.Context(), v, group, "1.1"))
			require.Equal(t, "1.1", v["app"].Version)
			require.Equal(t, "new-hash", v["app"].Artifacts["x86_64-linux"]["hash"])
			require.Equal(t, "amd64", v["app"].Artifacts["x86_64-linux"]["arch"])
			wantURL := "https://example.test/1.0.deb"
			if updateURL {
				wantURL = "https://example.test/1.1.deb"
			}
			require.Equal(t, wantURL, v["app"].Artifacts["x86_64-linux"]["url"])
		})
	}
}

func TestApplyBumpFailureIsAtomic(t *testing.T) {
	t.Parallel()
	v := versions{"app": {
		Version: "1.0",
		Artifacts: map[string]map[string]string{
			"aarch64-darwin": {"hash": "old-mac-hash", "url": "old-mac-url"},
			"x86_64-linux":   {"hash": "old-linux-hash", "url": "old-linux-url"},
		},
	}}
	group := versionGroup{targets: []target{{nixKey: "app", fetch: func(_ context.Context, system string, _ map[string]string, _ string) (artifact, error) {
		if system == "x86_64-linux" {
			return artifact{}, fmt.Errorf("download unavailable")
		}
		return artifact{sri: "new-hash", fields: map[string]string{"url": "new-url"}}, nil
	}}}}
	require.ErrorContains(t, applyBump(t.Context(), v, group, "1.1"), "download unavailable")
	require.Equal(t, "1.0", v["app"].Version)
	require.Equal(t, map[string]map[string]string{
		"aarch64-darwin": {"hash": "old-mac-hash", "url": "old-mac-url"},
		"x86_64-linux":   {"hash": "old-linux-hash", "url": "old-linux-url"},
	}, v["app"].Artifacts)
}
