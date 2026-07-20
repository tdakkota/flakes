# flakes

Personal Nix flake for AI coding CLIs and a couple of desktop apps that
aren't (well) packaged in nixpkgs. Everything is pinned by hand to a
specific upstream version/hash rather than tracking `latest`, and kept
current with `tools/bump` (see below).

## Packages

| Attribute | Aliases | What |
| --- | --- | --- |
| `claude-code` | `claude` | Claude Code CLI |
| `claude-desktop` | | Claude Desktop (Linux beta, repackaged official `.deb`) |
| `codex` | `default` | OpenAI Codex CLI |
| `grok` | | Grok CLI |
| `antigravity` | `agy` | Google Antigravity CLI |
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

`claude-code`'s Arch package is generated straight from the same
version/url/hash data used to build the Nix package (`mkPkgbuild` in
`flake.nix`), so the two can't drift apart. It isn't published to the AUR,
so `yay -S` won't find it - build and install it locally instead:

```sh
cd pkgbuilds/claude-code-bin
makepkg -si
```

Regenerate the PKGBUILD (e.g. after `tools/bump` updates `flake.nix`) with:

```sh
nix run .#gen-pkgbuilds
```

## Bumping versions

`tools/bump` is a small Go tool that checks each package's real upstream
feed (GitHub releases, `downloads.claude.ai`, an apt Packages index,
Proton's `version.json`, ...) and reports or applies version/hash updates
to `flake.nix`.

```sh
cd tools/bump
go run . -flake ../../flake.nix          # report only
go run . -flake ../../flake.nix -write   # apply and rewrite flake.nix
go run . -flake ../../flake.nix -only claude-code -write
go run . -flake ../../flake.nix -set grok=0.3.0 -write   # packages with no version feed
```

`grok` and `antigravity` have no discoverable upstream version feed and
are always reported as manual.
