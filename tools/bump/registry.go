package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-faster/errors"
)

// artifact is what a per-system fetch produces: the exact Nix SRI hash
// string to write into that system's "hash" field in versions.nix.
type artifact struct {
	sri string
}

// fetchArtifactFunc builds and hashes the artifact for one Nix system at a
// given version. fields carries that system's other current artifact
// fields (platform/target/arch/... - whatever versions.nix already has for
// it), which is how per-package URL shape stays out of the Go code below.
type fetchArtifactFunc func(ctx context.Context, system string, fields map[string]string, version string) (artifact, error)

// target is one top-level entry this group writes into versions.nix. Most
// groups have exactly one; vibe/vibe-acp share a version but are separate
// entries (separate Nix packages, separate artifact shapes).
type target struct {
	nixKey string
	fetch  fetchArtifactFunc
}

// versionGroup is one thing tools/bump checks: an upstream version feed
// (nil if there isn't one - bump via -set instead) plus the versions.nix
// entries it drives.
type versionGroup struct {
	name string
	// fetchLatest is nil for upstreams with no discoverable version feed;
	// those can only be bumped via -set.
	fetchLatest func(ctx context.Context) (string, error)
	targets     []target
}

// fetchDownload hashes the raw downloaded file, matching what Nix's
// `fetchurl` fixed-output hash covers - used by mkBinaryPackage and
// mkTarballPackage.
func fetchDownload(ctx context.Context, algo, url string) (artifact, error) {
	hex, err := downloadAndHash(ctx, url, algo)
	if err != nil {
		return artifact{}, err
	}
	sri, err := sriFromHex(algo, hex)
	if err != nil {
		return artifact{}, err
	}
	return artifact{sri: sri}, nil
}

// fetchZip hashes the *unpacked* contents of a zip file, matching what
// Nix's `fetchzip` (with stripRoot = false, as mkZipPackage uses) hashes -
// a NAR hash of the extracted tree, not the raw zip bytes. There's no
// pure-Go way to reproduce that, so this shells out to the same `nix`
// binary that will ultimately build the package.
func fetchZip(ctx context.Context, url string) (artifact, error) {
	sri, err := fetchZipNarHash(ctx, url)
	if err != nil {
		return artifact{}, err
	}
	return artifact{sri: sri}, nil
}

// registry mirrors the package definitions in versions.nix/flake.nix. URL
// templates are hand-kept in sync with flake.nix: there's no way to derive
// them generically since every upstream names things differently.
func registry() []versionGroup {
	return []versionGroup{
		{
			name: "grok",
			// Same plaintext channel-pointer endpoint https://x.ai/cli/install.sh reads
			// (BASE_URL_PRIMARY/$GROK_CHANNEL, default channel "stable").
			fetchLatest: func(ctx context.Context) (string, error) {
				return httpGetString(ctx, "https://x.ai/cli/stable")
			},
			targets: []target{{
				nixKey: "grok",
				fetch: func(ctx context.Context, system string, fields map[string]string, version string) (artifact, error) {
					url := fmt.Sprintf("https://x.ai/cli/grok-%s-%s", version, fields["platform"])
					return fetchDownload(ctx, "sha256", url)
				},
			}},
		},
		{
			name: "codex",
			fetchLatest: func(ctx context.Context) (string, error) {
				return githubLatestTag(ctx, "openai/codex", "rust-v")
			},
			targets: []target{{
				nixKey: "codex",
				fetch: func(ctx context.Context, system string, fields map[string]string, version string) (artifact, error) {
					url := fmt.Sprintf(
						"https://github.com/openai/codex/releases/download/rust-v%s/codex-package-%s.tar.gz",
						version, fields["target"],
					)
					return fetchDownload(ctx, "sha256", url)
				},
			}},
		},
		{
			name: "claude-code",
			fetchLatest: func(ctx context.Context) (string, error) {
				return httpGetString(ctx, "https://downloads.claude.ai/claude-code-releases/latest")
			},
			targets: []target{{
				nixKey: "claude-code",
				fetch: func(ctx context.Context, system string, fields map[string]string, version string) (artifact, error) {
					url := fmt.Sprintf("https://downloads.claude.ai/claude-code-releases/%s/%s/claude", version, fields["platform"])
					return fetchDownload(ctx, "sha256", url)
				},
			}},
		},
		{
			name: "vibe",
			fetchLatest: func(ctx context.Context) (string, error) {
				return githubLatestTag(ctx, "mistralai/mistral-vibe", "v")
			},
			targets: []target{
				{
					nixKey: "vibe",
					fetch: func(ctx context.Context, system string, fields map[string]string, version string) (artifact, error) {
						url := fmt.Sprintf(
							"https://github.com/mistralai/mistral-vibe/releases/download/v%s/vibe-%s-%s.zip",
							version, fields["arch"], version,
						)
						return fetchZip(ctx, url)
					},
				},
				{
					nixKey: "vibe-acp",
					fetch: func(ctx context.Context, system string, fields map[string]string, version string) (artifact, error) {
						url := fmt.Sprintf(
							"https://github.com/mistralai/mistral-vibe/releases/download/v%s/vibe-acp-%s-%s.zip",
							version, fields["arch"], version,
						)
						return fetchZip(ctx, url)
					},
				},
			},
		},
		{
			name: "copilot",
			fetchLatest: func(ctx context.Context) (string, error) {
				return githubLatestTag(ctx, "github/copilot-cli", "v")
			},
			targets: []target{{
				nixKey: "copilot",
				fetch: func(ctx context.Context, system string, fields map[string]string, version string) (artifact, error) {
					url := fmt.Sprintf("https://github.com/github/copilot-cli/releases/download/v%s/copilot-%s.tar.gz", version, fields["platform"])
					return fetchDownload(ctx, "sha256", url)
				},
			}},
		},
		{
			name: "opencode",
			fetchLatest: func(ctx context.Context) (string, error) {
				return githubLatestTag(ctx, "anomalyco/opencode", "v")
			},
			targets: []target{{
				nixKey: "opencode",
				// darwin ships as .zip (mkZipPackage/fetchzip); linux ships as .tar.gz
				// (mkTarballPackage/fetchurl) - see flake.nix's opencode package.
				fetch: func(ctx context.Context, system string, fields map[string]string, version string) (artifact, error) {
					base := fmt.Sprintf("https://github.com/anomalyco/opencode/releases/download/v%s/opencode-%s", version, fields["platform"])
					if strings.HasSuffix(system, "darwin") {
						return fetchZip(ctx, base+".zip")
					}
					return fetchDownload(ctx, "sha256", base+".tar.gz")
				},
			}},
		},
		{
			name: "kimi",
			fetchLatest: func(ctx context.Context) (string, error) {
				return githubLatestTag(ctx, "MoonshotAI/kimi-cli", "")
			},
			targets: []target{{
				nixKey: "kimi",
				fetch: func(ctx context.Context, system string, fields map[string]string, version string) (artifact, error) {
					url := fmt.Sprintf(
						"https://github.com/MoonshotAI/kimi-cli/releases/download/%s/kimi-%s-%s.tar.gz",
						version, version, fields["platform"],
					)
					return fetchDownload(ctx, "sha256", url)
				},
			}},
		},
		{
			name: "claude-desktop",
			fetchLatest: func(ctx context.Context) (string, error) {
				s, err := aptLatest(ctx, "https://downloads.claude.ai/claude-desktop/apt/stable/dists/stable/main/binary-amd64/Packages", "claude-desktop")
				if err != nil {
					return "", err
				}
				return s["Version"], nil
			},
			targets: []target{{
				nixKey: "claude-desktop",
				fetch:  claudeDesktopArtifact,
			}},
		},
		{
			name: "proton-pass",
			fetchLatest: func(ctx context.Context) (string, error) {
				r, err := protonLatestStable(ctx, "https://proton.me/download/PassDesktop/linux/version.json")
				if err != nil {
					return "", err
				}
				return r.Version, nil
			},
			targets: []target{{
				nixKey: "proton-pass",
				fetch:  protonPassArtifact,
			}},
		},
	}
}

func claudeDesktopArtifact(ctx context.Context, system string, fields map[string]string, version string) (artifact, error) {
	arch := fields["arch"]
	indexURL := map[string]string{
		"amd64": "https://downloads.claude.ai/claude-desktop/apt/stable/dists/stable/main/binary-amd64/Packages",
		"arm64": "https://downloads.claude.ai/claude-desktop/apt/stable/dists/stable/main/binary-arm64/Packages",
	}[arch]

	stanzas, err := fetchAptPackages(ctx, indexURL)
	if err != nil {
		return artifact{}, err
	}
	want := fmt.Sprintf("pool/main/c/claude-desktop/claude-desktop_%s_%s.deb", version, arch)
	for _, s := range stanzas {
		if s["Package"] == "claude-desktop" && s["Filename"] == want {
			sri, err := sriFromHex("sha256", s["SHA256"])
			if err != nil {
				return artifact{}, err
			}
			return artifact{sri: sri}, nil
		}
	}
	return artifact{}, errors.Errorf("claude-desktop %s (%s) not found in apt index", version, arch)
}

func protonPassArtifact(ctx context.Context, system string, fields map[string]string, version string) (artifact, error) {
	versionJSONURL := "https://proton.me/download/PassDesktop/linux/version.json"
	if strings.HasSuffix(system, "darwin") {
		versionJSONURL = "https://proton.me/download/PassDesktop/macos/version.json"
	}

	r, err := protonLatestStable(ctx, versionJSONURL)
	if err != nil {
		return artifact{}, err
	}
	if r.Version != version {
		return artifact{}, errors.Errorf("requested version %s but feed's latest Stable is %s", version, r.Version)
	}
	if len(r.File) == 0 {
		return artifact{}, errors.Errorf("no File entries for proton-pass %s", version)
	}
	sri, err := sriFromHex("sha512", r.File[0].Sha512CheckSum)
	if err != nil {
		return artifact{}, err
	}
	return artifact{sri: sri}, nil
}
