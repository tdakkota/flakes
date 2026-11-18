{
  lib,
  mkPkgbuild,
  version,
  artifacts,
}:
let
  archMap = {
    x86_64 = "x86_64-linux";
    aarch64 = "aarch64-linux";
  };
  ruleLists = import ./rule-lists.nix;
  hex =
    hash:
    builtins.convertHash {
      inherit hash;
      toHashFormat = "base16";
      hashAlgo = "sha256";
    };
in
mkPkgbuild {
  pname = "throne";
  inherit version;
  description = "Qt proxy client with sing-box, Xray and thronectl";
  homepage = "https://github.com/throneproj/Throne";
  license = "GPL-3.0-or-later";
  artifacts = lib.mapAttrs (_: system: {
    url = artifacts.${system}.url;
    sha256Hex = hex artifacts.${system}.hash;
  }) archMap;
  extraAttrs = ''
    depends=('qt6-base' 'qt6-tools' 'hicolor-icon-theme' 'iproute2' 'iptables' 'procps-ng' 'systemd' 'polkit')
    makedepends=('cmake' 'ninja')
    noextract=("$pkgname-$pkgver-$CARCH")
    source=("throne-$pkgver.tar.gz::https://github.com/throneproj/Throne/archive/refs/tags/$pkgver.tar.gz"
            "srslist.h::${ruleLists.url}")
    sha256sums=('${hex artifacts.x86_64-linux.sourceHash}' '${hex ruleLists.hash}')
  '';
  functions = ''
    prepare() {
      mkdir -p "$srcdir/vendor"
      bsdtar -xOf "$srcdir/$pkgname-$pkgver-$CARCH" 'data.tar.*' | bsdtar -xf - --no-same-owner -C "$srcdir/vendor"
      cp "$srcdir/srslist.h" "$srcdir/Throne-$pkgver/srslist.h"
    }

    build() {
      INPUT_VERSION="$pkgver" cmake -S "$srcdir/Throne-$pkgver" -B "$srcdir/build" -G Ninja \
        -DCMAKE_BUILD_TYPE=Release -DNKR_PACKAGE=ON -DNKR_CORE_IN_PATH=ON \
        -DNKR_DESKTOP_EXEC=Throne -DNKR_ELEVATION_HINT='Configure core capabilities explicitly for TUN mode.' \
        -DCMAKE_CXX_FLAGS=-Wno-error=maybe-uninitialized
      cmake --build "$srcdir/build"
    }
  '';
  packageBody = ''
    install -dm755 "$pkgdir/usr/lib/throne" "$pkgdir/usr/bin"
    cp -a "$srcdir/vendor/opt/Throne/." "$pkgdir/usr/lib/throne/"
    install -m755 "$srcdir/build/Throne" "$pkgdir/usr/lib/throne/Throne"
    chmod 755 "$pkgdir/usr/lib/throne/ThroneCore" "$pkgdir/usr/lib/throne/thronectl"
    ln -s /usr/lib/throne/Throne "$pkgdir/usr/bin/Throne"
    ln -s Throne "$pkgdir/usr/bin/throne"
    ln -s /usr/lib/throne/thronectl "$pkgdir/usr/bin/thronectl"
    install -Dm644 "$pkgdir/usr/lib/throne/Throne.png" "$pkgdir/usr/share/icons/hicolor/512x512/apps/Throne.png"
    install -Dm644 /dev/stdin "$pkgdir/usr/share/applications/Throne.desktop" <<'DESKTOP'
    [Desktop Entry]
    Type=Application
    Name=Throne
    Exec=Throne
    Icon=Throne
    Categories=Network;
    Terminal=false
    DESKTOP
  '';
}
