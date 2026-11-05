{
  pkgs,
  version,
  artifact,
}:
let
  inherit (pkgs) lib;
  release = pkgs.fetchurl {
    inherit (artifact) url hash;
  };
  ruleLists = pkgs.fetchurl {
    url = "https://raw.githubusercontent.com/throneproj/routeprofiles/1aa995b5fc30c26b931a61171aea2ab49c3d1711/srslist.h";
    hash = "sha256-jKYUjgxpE1zwcVv+9Fdj8H1QnPZpTzmzVsMfOMOKGFw=";
  };
in
pkgs.stdenv.mkDerivation {
  pname = "throne";
  inherit version;
  src = pkgs.fetchurl {
    url = "https://github.com/throneproj/Throne/archive/refs/tags/${version}.tar.gz";
    hash = artifact.sourceHash;
  };

  __structuredAttrs = true;
  strictDeps = true;
  nativeBuildInputs = with pkgs; [
    autoPatchelfHook
    cmake
    dpkg
    ninja
    qt6.wrapQtAppsHook
  ];
  buildInputs = with pkgs; [
    stdenv.cc.cc.lib
    qt6.qtbase
    qt6.qttools
  ];
  dontWrapQtApps = true;
  env.INPUT_VERSION = version;
  env.NIX_CFLAGS_COMPILE = "-Wno-error=maybe-uninitialized";
  cmakeFlags = [
    (lib.cmakeBool "NKR_PACKAGE" true)
    (lib.cmakeBool "NKR_CORE_IN_PATH" true)
    (lib.cmakeFeature "NKR_DESKTOP_EXEC" "Throne")
    (lib.cmakeFeature "NKR_ELEVATION_HINT" "Use the flake's optional TUN setup or NixOS module.")
  ];
  preBuild = ''
    ln -s ${ruleLists} srslist.h
  '';

  installPhase = ''
    runHook preInstall
    releaseRoot="$TMPDIR/throne-release"
    mkdir -p "$releaseRoot" "$out/share/throne"
    dpkg-deb --fsys-tarfile ${release} | tar -x -C "$releaseRoot" --no-same-owner --no-same-permissions
    cp -a "$releaseRoot/opt/Throne/." "$out/share/throne/"
    chmod 0755 "$out/share/throne/ThroneCore" "$out/share/throne/thronectl"
    # Keep the real GUI basename and core together for ThroneCore's parent check.
    install -m755 Throne "$out/share/throne/Throne"
    makeQtWrapper "$out/share/throne/Throne" "$out/bin/Throne" \
      --prefix PATH : /run/wrappers/bin:/usr/local/libexec/throne:${
        lib.makeBinPath [
          pkgs.iproute2
          pkgs.iptables
          pkgs.procps
          pkgs.systemd
          pkgs.polkit
        ]
      }
    ln -s Throne "$out/bin/throne"
    ln -s "$out/share/throne/thronectl" "$out/bin/thronectl"
    install -Dm644 "$out/share/throne/Throne.png" \
      "$out/share/icons/hicolor/512x512/apps/Throne.png"
    mkdir -p "$out/share/applications"
    cat > "$out/share/applications/Throne.desktop" <<EOF
    [Desktop Entry]
    Type=Application
    Name=Throne
    Exec=$out/bin/Throne
    Icon=Throne
    Categories=Network;
    Terminal=false
    EOF
    runHook postInstall
  '';

  doInstallCheck = true;
  installCheckPhase = ''
    runHook preInstallCheck
    test -x "$out/bin/Throne"
    test -x "$out/share/throne/ThroneCore"
    test -x "$out/share/throne/thronectl"
    runHook postInstallCheck
  '';

  meta = {
    description = "Subscription-capable Qt proxy client powered by sing-box and Xray";
    homepage = "https://github.com/throneproj/Throne";
    changelog = "https://github.com/throneproj/Throne/releases/tag/${version}";
    license = lib.licenses.gpl3Plus;
    mainProgram = "Throne";
    platforms = [
      "x86_64-linux"
      "aarch64-linux"
    ];
    sourceProvenance = with lib.sourceTypes; [
      fromSource
      binaryNativeCode
    ];
  };
}
