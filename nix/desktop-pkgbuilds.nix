{
  lib,
  versions,
  mkPkgbuild,
}:
let
  archMap = {
    x86_64 = "x86_64-linux";
    aarch64 = "aarch64-linux";
  };
  electronDepends = [
    "alsa-lib"
    "at-spi2-core"
    "cairo"
    "cups"
    "dbus"
    "expat"
    "gcc-libs"
    "glib2"
    "glibc"
    "gtk3"
    "libdrm"
    "libx11"
    "libxcb"
    "libxcomposite"
    "libxdamage"
    "libxext"
    "libxfixes"
    "libxkbcommon"
    "libxrandr"
    "libxtst"
    "mesa"
    "nspr"
    "nss"
    "pango"
    "systemd"
    "xdg-utils"
  ];
  mkDeb =
    {
      pname,
      description,
      homepage,
      depends,
      urlFor,
      checksum ? "sha256",
    }:
    let
      data = versions.${pname};
    in
    mkPkgbuild {
      inherit
        pname
        description
        homepage
        checksum
        ;
      inherit (data) version;
      artifacts = lib.mapAttrs (arch: system: {
        url = urlFor data.version data.artifacts.${system};
        "${checksum}Hex" = builtins.convertHash {
          hash = data.artifacts.${system}.hash;
          toHashFormat = "base16";
          hashAlgo = checksum;
        };
      }) (lib.filterAttrs (_: system: data.artifacts ? ${system}) archMap);
      extraAttrs = ''
        depends=(${lib.concatMapStringsSep " " lib.escapeShellArg (electronDepends ++ depends)})
        noextract=("$pkgname-$pkgver-$CARCH")
      '';
      packageBody = ''
        bsdtar -xOf "$srcdir/$pkgname-$pkgver-$CARCH" 'data.tar.*' | bsdtar -xf - --no-same-owner -C "$pkgdir" ./usr
      '';
    };
in
{
  claude-desktop = mkDeb {
    pname = "claude-desktop";
    description = "Claude Desktop for Linux (official binary)";
    homepage = "https://claude.ai";
    depends = [
      "libcap-ng"
      "libseccomp"
      "libnotify"
      "libsecret"
      "libpulse"
      "libayatana-appindicator"
      "util-linux-libs"
      "libpipewire"
      "xdg-desktop-portal"
      "xdg-desktop-portal-gtk"
    ];
    urlFor =
      version: artifact:
      "https://downloads.claude.ai/claude-desktop/apt/stable/pool/main/c/claude-desktop/claude-desktop_${version}_${artifact.arch}.deb";
  };
  proton-pass = mkDeb {
    pname = "proton-pass";
    description = "Proton Pass desktop application (official binary)";
    homepage = "https://proton.me/pass";
    depends = [
      "libnotify"
      "libsecret"
    ];
    checksum = "sha512";
    urlFor = _: artifact: artifact.url;
  };
}
