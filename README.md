# flakes

Personal Nix flake for AI coding CLIs and a couple of desktop apps that
aren't (well) packaged in nixpkgs. Everything is pinned in `versions.nix`
to a specific upstream version/hash rather than tracking `latest`, and
kept current with `tools/bump` (see below).

## Packages

| Attribute | Aliases | What |
| --- | --- | --- |
| `claude-code` | `claude` | Claude Code CLI |
| `claude-desktop` | | Claude Desktop (Linux beta, repackaged official `.deb`) |
| `codex` | `default` | OpenAI Codex CLI |
| `grok` | | Grok CLI |
| `vibe` | | Mistral Vibe CLI |
| `vibe-acp` | | Mistral Vibe ACP |
| `copilot` | | GitHub Copilot CLI |
| `opencode` | | opencode CLI |
| `kimi` | | Kimi (Moonshot AI) CLI |
| `proton-pass` | | Proton Pass desktop app, built from Proton's own release feed |

Supported systems: `x86_64-linux`, `aarch64-linux`, `aarch64-darwin`.
`claude-desktop` is Linux-only (no upstream macOS build); `proton-pass` is
`x86_64-linux` + `aarch64-darwin` only (no upstream `aarch64-linux` build).

```sh
nix run .#claude-code
nix profile install .#codex
```

## Installing on Arch Linux

Every CLI package (all but `claude-desktop` and `proton-pass`) also has an
Arch `PKGBUILD`, generated from the same version/url/hash data used to
build the Nix package (`mkPkgbuild` in `flake.nix`) so the two can't drift
apart. None of these are published to the AUR, so `yay -S` won't find
them - build and install locally instead:

```sh
cd pkgbuilds/claude-code-bin   # or grok-cli-bin, codex-cli-bin, copilot-cli-bin,
                                # opencode-bin, kimi-cli-bin, mistral-vibe-bin,
                                # mistral-vibe-acp-bin
makepkg -si
```

Regenerate all of them (e.g. after `tools/bump` updates `versions.nix`) with:

```sh
nix run .#gen-pkgbuilds
```

## Bumping versions

`tools/bump` is a small Go tool that checks each package's real upstream
feed (GitHub releases, `downloads.claude.ai`, an apt Packages index,
Proton's `version.json`, ...) and reports or applies version/hash updates
to `versions.nix` - it regenerates the whole file rather than patching it
in place.

```sh
cd tools/bump
go run .                                    # report only
go run . -write                             # apply and rewrite versions.nix
go run . -only claude-code -write
go run . -set grok=0.3.0 -write             # packages with no version feed
```
