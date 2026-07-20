# flakes

Personal Nix flake packaging AI coding CLIs and a couple of desktop apps
not (well) packaged in nixpkgs. See README.md for the package list.

## Commands

- Format: `nix run nixpkgs#nixfmt -- flake.nix` (NOT `nix fmt` — it misbehaves in this repo)
- Check: `nix flake check --no-build`
- Build one package: `nix build .#<name> -L`
- Regenerate PKGBUILDs: `nix run .#gen-pkgbuilds`
- Check/apply version bumps: `cd tools/bump && go run . -flake ../../flake.nix [-write]`

Most `nix` invocations need `dangerouslyDisableSandbox` in this
environment: the sandbox links dotfiles to `/dev/null`, which breaks Nix's
git-tree evaluation, and its network allowlist blocks everything nix needs
to fetch except `githubusercontent.com`.

## Architecture

- `flake.nix` has three generic builder helpers for simple packages:
  `mkBinaryPackage` (single downloaded executable), `mkTarballPackage`,
  `mkZipPackage`. `claude-desktop` and `proton-pass` don't fit these (they
  need `autoPatchelfHook`/`dpkg`/`asar` handling) and are bespoke
  `stdenv.mkDerivation`s instead.
- `mkPkgbuild` (also in `flake.nix`) renders Arch `PKGBUILD`s from the same
  version/url/hash data used to build the Nix package, so the two can't
  drift apart. Currently only wired up for `claude-code`
  (`pkgbuilds/claude-code-bin/`).
- `tools/bump` is a standalone Go module (own `go.mod`) that reads each
  package's pinned version/hash from `flake.nix` and checks its real
  upstream feed (GitHub releases, an apt Packages index, Proton's
  `version.json`, etc.), optionally patching `flake.nix` in place.

## CI

- `.github/workflows/ci.yml`: `nix flake check --no-build`, `nixfmt
  --check`, and Go build/vet/fmt on every push/PR.
- `.github/workflows/bump.yml`: runs `tools/bump -write` daily and opens a
  PR (branch `auto/bump-versions`) when there's a bump — doesn't autocommit
  directly like ogen's `tidy-autocommit.yml` does for dependabot's go.mod
  PRs, since there's no dependabot PR to attach to here (dependabot has no
  Nix ecosystem support, so it can't see these pinned CLI versions at all).
- `.github/dependabot.yml` only covers `tools/bump`'s `go.mod` and the
  workflow actions themselves — hence `tools/bump` for everything else.

## Gotchas

- Packages built with `mkZipPackage` (`vibe`, `vibe-acp`, `opencode` on
  darwin) use `pkgs.fetchzip`, which hashes the *unpacked* directory tree
  (NAR hash), not the raw zip bytes. Don't hash the downloaded file
  directly when updating these by hand — use
  `nix store prefetch-file --unpack --json <url>` instead (this is what
  `tools/bump` does internally).
- `grok` and `antigravity` have no discoverable upstream version feed
  (`antigravity`'s download URL embeds an opaque build-id not derivable
  from the version); `tools/bump` can't auto-check these.
