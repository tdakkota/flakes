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
| `throne` | | Throne GUI with subscription support, sing-box/Xray core and `thronectl` |
| `throne-tun-setup` | | Optional root installer for Throne's TUN capability wrapper on non-NixOS Linux |

Supported systems: `x86_64-linux`, `aarch64-linux`, `aarch64-darwin`.
`claude-desktop` is Linux-only (no upstream macOS build); `proton-pass` is
`x86_64-linux` + `aarch64-darwin` only (no upstream `aarch64-linux` build).
Throne and its optional TUN setup support `x86_64-linux` and `aarch64-linux`.

```sh
nix run .#claude-code
nix profile install .#codex
```

## Installing on Arch Linux

The upstream CLI packages (excluding the desktop apps and TUN installer) also have an
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

The daily GitHub workflow checks all packages in one job and batches updates
into one PR on `auto/bump-versions`. Subsequent runs refresh the same open PR;
after it is merged, new updates open a new PR. Runs are serialized to avoid
concurrent writes to the shared branch. One unavailable upstream artifact does
not block successful updates from other packages, but the run still reports
failure. A failed check with no changes leaves the existing PR untouched.
Update PRs mention, assign, and request review from
`@tdakkota`. Enable "Allow GitHub Actions to create and approve pull requests"
in the repository's Actions settings so its token can create these PRs.
Vibe and Vibe ACP track releases independently and select only stable releases
that publish every supported platform's binary.

## Throne and TUN permissions

```sh
nix build .#throne
nix run .#throne
```

The GUI uses a writable user configuration directory. Only its core needs
network privileges for TUN. A normal `nix profile install` cannot grant them:
the Nix store is immutable, and the NixOS `programs.throne` module does not
configure an Arch host. Proxy mode works without the optional TUN setup.

### Arch and other Linux distributions

Build and review the optional installer separately:

```sh
nix build .#throne-tun-setup -o result-throne-tun
sudo ./result-throne-tun/bin/throne-tun-setup --user "$USER"
```

This explicit root operation installs NixOS's static security wrapper at
`/usr/local/libexec/throne/ThroneCore`, owned by root and executable only by
the `throne-tun` group. It grants the core `CAP_NET_ADMIN`, `CAP_NET_RAW`, and
`CAP_NET_BIND_SERVICE` through ambient capabilities. The wrapper's own
`CAP_SETPCAP` is not passed to the fixed core from the store.

The installer adds the named user to that group, creates a root-owned Nix GC
root, and installs a group-scoped systemd-resolved polkit rule. Reboot before
launching from desktop/user services, or run directly from a fresh
`newgrp throne-tun` shell. It does not start a daemon, connect a profile, or
change routes/DNS. Repeat it after upgrading Throne to retarget the wrapper.
The GUI stays unprivileged, and no file in `/nix/store` is chmodded or setcap'd.
Process inspection capabilities (`SYS_PTRACE`/`DAC_READ_SEARCH`) are not granted.

To undo the optional setup, remove the wrapper, GC root, and polkit rule,
then remove the user from `throne-tun`. Do not remove the Nix store package
while its core is running.

### NixOS

Import this flake's module alongside your normal configuration:

```nix
{
  imports = [ inputs.flakes.nixosModules.throne ];
  programs.throne = {
    enable = true;
    tunMode = {
      enable = true;
      allowedUsers = [ "tdakkota" ];
    };
  };
}
```

The module selects this flake's package and installs the group-restricted
`ThroneCore` wrapper. It replaces the older Nixpkgs `throne-core` wrapper name.

Throne's GUI is built from the pinned release source with its supported
PATH-based core lookup. The matching prebuilt core is retained next to the
real `Throne` executable, preserving upstream's parent-process check. Release
URLs, core hashes, and source hashes are maintained by `tools/bump`.
