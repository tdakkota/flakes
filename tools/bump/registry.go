package main

import (
	"context"
	"fmt"

	"github.com/go-faster/errors"
)

// artifact is what a per-system fetch produces: the exact Nix SRI hash
// string to patch into a `hash = "...";` field in flake.nix.
type artifact struct {
	sri string
}

// artifactTable mirrors one `xArtifacts = { system = { ...; hash }; };`
// block in flake.nix.
type artifactTable struct {
	nixVar  string
	systems map[string]func(ctx context.Context, version string) (artifact, error)
}

// versionGroup mirrors one `xVersion` pin (and everything derived from it -
// possibly several artifact tables, e.g. vibe + vibe-acp share a version).
type versionGroup struct {
	name          string
	nixVersionVar string
	// fetchLatest is nil for upstreams with no discoverable version feed
	// (grok, antigravity); those can only be bumped via -set.
	fetchLatest func(ctx context.Context) (string, error)
	tables      []artifactTable
}

// downloadArtifact hashes the raw downloaded file, matching what Nix's
// `fetchurl` fixed-output hash covers - used by mkBinaryPackage and
// mkTarballPackage.
func downloadArtifact(algo, url string) func(ctx context.Context, version string) (artifact, error) {
	return func(ctx context.Context, version string) (artifact, error) {
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
}

// zipArtifact hashes the *unpacked* contents of a zip file, matching what
// Nix's `fetchzip` (with stripRoot = false, as mkZipPackage uses) hashes -
// a NAR hash of the extracted tree, not the raw zip bytes. There's no pure-Go
// way to reproduce that, so this shells out to the same `nix` binary that
// will ultimately build the package.
func zipArtifact(url string) func(ctx context.Context, version string) (artifact, error) {
	return func(ctx context.Context, version string) (artifact, error) {
		sri, err := fetchZipNarHash(ctx, url)
		if err != nil {
			return artifact{}, err
		}
		return artifact{sri: sri}, nil
	}
}

// registry mirrors the package definitions in flake.nix. URL templates and
// per-system platform strings are hand-kept in sync with it: there is no way
// to derive them generically since every upstream names things differently.
func registry() []versionGroup {
	return []versionGroup{
		{
			name:          "grok",
			nixVersionVar: "grokVersion",
			// Same plaintext channel-pointer endpoint https://x.ai/cli/install.sh reads
			// (BASE_URL_PRIMARY/$GROK_CHANNEL, default channel "stable").
			fetchLatest: func(ctx context.Context) (string, error) {
				return httpGetString(ctx, "https://x.ai/cli/stable")
			},
			tables: []artifactTable{{
				nixVar: "grokArtifacts",
				systems: map[string]func(context.Context, string) (artifact, error){
					"x86_64-linux": func(ctx context.Context, v string) (artifact, error) {
						return downloadArtifact("sha256", fmt.Sprintf("https://x.ai/cli/grok-%s-linux-x86_64", v))(ctx, v)
					},
					"aarch64-linux": func(ctx context.Context, v string) (artifact, error) {
						return downloadArtifact("sha256", fmt.Sprintf("https://x.ai/cli/grok-%s-linux-aarch64", v))(ctx, v)
					},
					"aarch64-darwin": func(ctx context.Context, v string) (artifact, error) {
						return downloadArtifact("sha256", fmt.Sprintf("https://x.ai/cli/grok-%s-macos-aarch64", v))(ctx, v)
					},
				},
			}},
		},
		{
			name:          "antigravity",
			nixVersionVar: "antigravityVersion",
			fetchLatest:   nil, // download URL embeds an opaque build-id suffix that isn't derivable from the version alone
			tables:        nil,
		},
		{
			name:          "codex",
			nixVersionVar: "codexVersion",
			fetchLatest: func(ctx context.Context) (string, error) {
				return githubLatestTag(ctx, "openai/codex", "rust-v")
			},
			tables: []artifactTable{{
				nixVar: "codexArtifacts",
				systems: map[string]func(context.Context, string) (artifact, error){
					"x86_64-linux": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://github.com/openai/codex/releases/download/rust-v%s/codex-package-x86_64-unknown-linux-musl.tar.gz", v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
					"aarch64-linux": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://github.com/openai/codex/releases/download/rust-v%s/codex-package-aarch64-unknown-linux-musl.tar.gz", v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
					"aarch64-darwin": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://github.com/openai/codex/releases/download/rust-v%s/codex-package-aarch64-apple-darwin.tar.gz", v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
				},
			}},
		},
		{
			name:          "claude-code",
			nixVersionVar: "claudeCodeVersion",
			fetchLatest: func(ctx context.Context) (string, error) {
				return httpGetString(ctx, "https://downloads.claude.ai/claude-code-releases/latest")
			},
			tables: []artifactTable{{
				nixVar: "claudeCodeArtifacts",
				systems: map[string]func(context.Context, string) (artifact, error){
					"x86_64-linux": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://downloads.claude.ai/claude-code-releases/%s/linux-x64/claude", v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
					"aarch64-linux": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://downloads.claude.ai/claude-code-releases/%s/linux-arm64/claude", v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
					"aarch64-darwin": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://downloads.claude.ai/claude-code-releases/%s/darwin-arm64/claude", v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
				},
			}},
		},
		{
			name:          "vibe",
			nixVersionVar: "vibeVersion",
			fetchLatest: func(ctx context.Context) (string, error) {
				return githubLatestTag(ctx, "mistralai/mistral-vibe", "v")
			},
			tables: []artifactTable{
				{
					nixVar: "vibeArtifacts",
					systems: map[string]func(context.Context, string) (artifact, error){
						"x86_64-linux": func(ctx context.Context, v string) (artifact, error) {
							url := fmt.Sprintf("https://github.com/mistralai/mistral-vibe/releases/download/v%s/vibe-linux-x86_64-%s.zip", v, v)
							return zipArtifact(url)(ctx, v)
						},
						"aarch64-linux": func(ctx context.Context, v string) (artifact, error) {
							url := fmt.Sprintf("https://github.com/mistralai/mistral-vibe/releases/download/v%s/vibe-linux-aarch64-%s.zip", v, v)
							return zipArtifact(url)(ctx, v)
						},
						"aarch64-darwin": func(ctx context.Context, v string) (artifact, error) {
							url := fmt.Sprintf("https://github.com/mistralai/mistral-vibe/releases/download/v%s/vibe-darwin-aarch64-%s.zip", v, v)
							return zipArtifact(url)(ctx, v)
						},
					},
				},
				{
					nixVar: "vibeAcpArtifacts",
					systems: map[string]func(context.Context, string) (artifact, error){
						"x86_64-linux": func(ctx context.Context, v string) (artifact, error) {
							url := fmt.Sprintf("https://github.com/mistralai/mistral-vibe/releases/download/v%s/vibe-acp-linux-x86_64-%s.zip", v, v)
							return zipArtifact(url)(ctx, v)
						},
						"aarch64-linux": func(ctx context.Context, v string) (artifact, error) {
							url := fmt.Sprintf("https://github.com/mistralai/mistral-vibe/releases/download/v%s/vibe-acp-linux-aarch64-%s.zip", v, v)
							return zipArtifact(url)(ctx, v)
						},
						"aarch64-darwin": func(ctx context.Context, v string) (artifact, error) {
							url := fmt.Sprintf("https://github.com/mistralai/mistral-vibe/releases/download/v%s/vibe-acp-darwin-aarch64-%s.zip", v, v)
							return zipArtifact(url)(ctx, v)
						},
					},
				},
			},
		},
		{
			name:          "copilot",
			nixVersionVar: "copilotVersion",
			fetchLatest: func(ctx context.Context) (string, error) {
				return githubLatestTag(ctx, "github/copilot-cli", "v")
			},
			tables: []artifactTable{{
				nixVar: "copilotArtifacts",
				systems: map[string]func(context.Context, string) (artifact, error){
					"x86_64-linux": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://github.com/github/copilot-cli/releases/download/v%s/copilot-linux-x64.tar.gz", v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
					"aarch64-linux": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://github.com/github/copilot-cli/releases/download/v%s/copilot-linux-arm64.tar.gz", v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
					"aarch64-darwin": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://github.com/github/copilot-cli/releases/download/v%s/copilot-darwin-arm64.tar.gz", v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
				},
			}},
		},
		{
			name:          "opencode",
			nixVersionVar: "opencodeVersion",
			fetchLatest: func(ctx context.Context) (string, error) {
				return githubLatestTag(ctx, "anomalyco/opencode", "v")
			},
			tables: []artifactTable{{
				nixVar: "opencodeArtifacts",
				systems: map[string]func(context.Context, string) (artifact, error){
					"x86_64-linux": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://github.com/anomalyco/opencode/releases/download/v%s/opencode-linux-x64.tar.gz", v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
					"aarch64-linux": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://github.com/anomalyco/opencode/releases/download/v%s/opencode-linux-arm64.tar.gz", v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
					"aarch64-darwin": func(ctx context.Context, v string) (artifact, error) {
						// opencode ships darwin as a .zip (mkZipPackage/fetchzip), unlike its .tar.gz linux builds.
						url := fmt.Sprintf("https://github.com/anomalyco/opencode/releases/download/v%s/opencode-darwin-arm64.zip", v)
						return zipArtifact(url)(ctx, v)
					},
				},
			}},
		},
		{
			name:          "kimi",
			nixVersionVar: "kimiVersion",
			fetchLatest: func(ctx context.Context) (string, error) {
				return githubLatestTag(ctx, "MoonshotAI/kimi-cli", "")
			},
			tables: []artifactTable{{
				nixVar: "kimiArtifacts",
				systems: map[string]func(context.Context, string) (artifact, error){
					"x86_64-linux": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://github.com/MoonshotAI/kimi-cli/releases/download/%s/kimi-%s-x86_64-unknown-linux-gnu.tar.gz", v, v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
					"aarch64-linux": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://github.com/MoonshotAI/kimi-cli/releases/download/%s/kimi-%s-aarch64-unknown-linux-gnu.tar.gz", v, v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
					"aarch64-darwin": func(ctx context.Context, v string) (artifact, error) {
						url := fmt.Sprintf("https://github.com/MoonshotAI/kimi-cli/releases/download/%s/kimi-%s-aarch64-apple-darwin.tar.gz", v, v)
						return downloadArtifact("sha256", url)(ctx, v)
					},
				},
			}},
		},
		{
			name:          "claude-desktop",
			nixVersionVar: "claudeDesktopVersion",
			fetchLatest: func(ctx context.Context) (string, error) {
				s, err := aptLatest(ctx, "https://downloads.claude.ai/claude-desktop/apt/stable/dists/stable/main/binary-amd64/Packages", "claude-desktop")
				if err != nil {
					return "", err
				}
				return s["Version"], nil
			},
			tables: []artifactTable{{
				nixVar: "claudeDesktopArtifacts",
				systems: map[string]func(context.Context, string) (artifact, error){
					"x86_64-linux":  claudeDesktopArtifact("amd64"),
					"aarch64-linux": claudeDesktopArtifact("arm64"),
				},
			}},
		},
		{
			name:          "proton-pass",
			nixVersionVar: "protonPassVersion",
			fetchLatest: func(ctx context.Context) (string, error) {
				r, err := protonLatestStable(ctx, "https://proton.me/download/PassDesktop/linux/version.json")
				if err != nil {
					return "", err
				}
				return r.Version, nil
			},
			tables: []artifactTable{{
				nixVar: "protonPassArtifacts",
				systems: map[string]func(context.Context, string) (artifact, error){
					"x86_64-linux":   protonPassArtifact("https://proton.me/download/PassDesktop/linux/version.json"),
					"aarch64-darwin": protonPassArtifact("https://proton.me/download/PassDesktop/macos/version.json"),
				},
			}},
		},
	}
}

func claudeDesktopArtifact(arch string) func(ctx context.Context, version string) (artifact, error) {
	binaryURL := map[string]string{
		"amd64": "https://downloads.claude.ai/claude-desktop/apt/stable/dists/stable/main/binary-amd64/Packages",
		"arm64": "https://downloads.claude.ai/claude-desktop/apt/stable/dists/stable/main/binary-arm64/Packages",
	}[arch]

	return func(ctx context.Context, version string) (artifact, error) {
		stanzas, err := fetchAptPackages(ctx, binaryURL)
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
}

func protonPassArtifact(versionJSONURL string) func(ctx context.Context, version string) (artifact, error) {
	return func(ctx context.Context, version string) (artifact, error) {
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
}
