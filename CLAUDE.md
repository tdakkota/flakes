# flakes

Personal Nix flake packaging AI coding CLIs and a couple of desktop apps
not (well) packaged in nixpkgs. See README.md for the package list.

**When changing what's packaged, how it's installed, or the bump/PKGBUILD
tooling, update README.md too** - it's the user-facing doc and easy to
forget since most of this file's own content lives in `flake.nix`.

## Commands

- Format: `nix run nixpkgs#nixfmt -- flake.nix versions.nix` (NOT `nix fmt` — it misbehaves in this repo)
- Check: `nix flake check --no-build`
- Build one package: `nix build .#<name> -L`
- Regenerate all PKGBUILDs: `nix run .#gen-pkgbuilds`
- Check/apply version bumps: `cd tools/bump && go run . [-write]`

Most `nix` invocations need `dangerouslyDisableSandbox` in this
environment: the sandbox links dotfiles to `/dev/null`, which breaks Nix's
git-tree evaluation, and its network allowlist blocks everything nix needs
to fetch except `githubusercontent.com`.

## Architecture

- `versions.nix` holds every package's pinned version + per-system
  artifact fields (platform/target/arch/url/hash - whatever that
  package's fetch needs). It's generated wholesale by `tools/bump`, not
  hand-edited - `flake.nix` just does `versions = import ./versions.nix;`
  and reads from it. This replaced an earlier design that regex-patched
  `flake.nix` text directly, which was fragile; a separate generated data
  file the flake merely imports is far more robust.
- `flake.nix` has three generic builder helpers for simple packages:
  `mkBinaryPackage` (single downloaded executable), `mkTarballPackage`,
  `mkZipPackage`. `claude-desktop` and `proton-pass` don't fit these (they
  need `autoPatchelfHook`/`dpkg`/`asar` handling) and are bespoke
  `stdenv.mkDerivation`s instead.
- `mkPkgbuild` (also in `flake.nix`) renders Arch `PKGBUILD`s from the
  same version/url/hash data used to build the Nix package, so the two
  can't drift apart. Wired up for every CLI package except
  `claude-desktop`/`proton-pass` (GUI apps, out of scope for the current
  simple-binary/tarball template). `vibe`/`vibe-acp` are the exception:
  their pinned hash is a NAR hash (see Gotchas), which pacman can't use,
  so their PKGBUILD ships a `__HASH_X86_64__`/`__HASH_AARCH64__`
  placeholder that `gen-pkgbuilds` resolves by fetching+hashing the raw
  archive itself at generation time.
- `tools/bump` is a standalone Go module (own `go.mod`) that reads
  `versions.nix` via `nix eval --json --file`, checks each package's real
  upstream feed (GitHub releases, an apt Packages index, Proton's
  `version.json`, etc.), and - with `-write` - regenerates the whole file
  (via a small Nix-value renderer + `nixfmt`), never patches it in place.

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
  `tools/bump` does internally, and it's also why `vibe`/`vibe-acp`'s
  PKGBUILDs need the live-fetch step described above).
