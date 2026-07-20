{
  description = "A minimal flake template that you can adapt to your own project";

  inputs.nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0";

  outputs =
    { self, ... }@inputs:
    let
      inherit (inputs.nixpkgs) lib;

      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];

      forEachSupportedSystem =
        f:
        lib.genAttrs supportedSystems (
          system:
          f {
            inherit system;
            pkgs = import inputs.nixpkgs {
              inherit system;
              config.allowUnfree = true;
            };
          }
        );

      packageMeta = {
        license = lib.licenses.unfree;
        sourceProvenance = [ lib.sourceTypes.binaryNativeCode ];
      };

      mkBinaryPackage =
        pkgs:
        {
          pname,
          version,
          src,
          executableName ? pname,
          extraExecutables ? [ ],
          postInstall ? "",
          meta ? { },
        }:
        pkgs.stdenvNoCC.mkDerivation {
          inherit
            pname
            version
            src
            postInstall
            ;

          dontUnpack = true;
          dontStrip = true;

          installPhase = ''
            runHook preInstall

            install -Dm755 $src $out/bin/${executableName}
          ''
          + lib.concatMapStringsSep "\n" (name: ''
            ln -s ${executableName} $out/bin/${name}
          '') extraExecutables
          + ''

            runHook postInstall
          '';

          meta = packageMeta // meta;
        };

      mkTarballPackage =
        pkgs:
        {
          pname,
          version,
          src,
          binaryPath,
          executableName ? pname,
          meta ? { },
        }:
        pkgs.stdenvNoCC.mkDerivation {
          inherit pname version src;

          dontStrip = true;
          sourceRoot = ".";

          installPhase = ''
            runHook preInstall

            mkdir -p $out/share/${pname} $out/bin
            cp -R . $out/share/${pname}/
            chmod -R u+w $out/share/${pname}
            chmod 755 $out/share/${pname}/${binaryPath}
            ln -s $out/share/${pname}/${binaryPath} $out/bin/${executableName}

            runHook postInstall
          '';

          meta = packageMeta // meta;
        };

      mkZipPackage =
        pkgs:
        {
          pname,
          version,
          url,
          hash,
          binaryPath ? pname,
          executableName ? pname,
          meta ? { },
        }:
        pkgs.stdenvNoCC.mkDerivation {
          inherit pname version;
          src = pkgs.fetchzip {
            inherit url hash;
            stripRoot = false;
          };

          dontStrip = true;

          installPhase = ''
            runHook preInstall

            mkdir -p $out/share/${pname} $out/bin
            cp -R $src/. $out/share/${pname}/
            chmod -R u+w $out/share/${pname}
            chmod 755 $out/share/${pname}/${binaryPath}
            ln -s $out/share/${pname}/${binaryPath} $out/bin/${executableName}

            runHook postInstall
          '';

          meta = packageMeta // meta;
        };

      grokVersion = "0.2.106";
      grokArtifacts = {
        x86_64-linux = {
          platform = "linux-x86_64";
          hash = "sha256-cYDQ4DzCpJYDP/Oq4iI84jlEapgnpZ+qdgkcft1eHDg=";
        };
        aarch64-linux = {
          platform = "linux-aarch64";
          hash = "sha256-0SvhaY1W1FQ/HxCVwsJs09F6ZOiHcmKWc3QJkcGI5P8=";
        };
        aarch64-darwin = {
          platform = "macos-aarch64";
          hash = "sha256-cin14qabBYMshtuCvr2lQekrXCSVj7+s9cj0YzlNMCc=";
        };
      };

      codexVersion = "0.144.6";
      codexArtifacts = {
        x86_64-linux = {
          target = "x86_64-unknown-linux-musl";
          hash = "sha256-ma5I5HQ9psUw7NmYqy9+ZlcsCS9BkMiNyoI2wHsGzh0=";
        };
        aarch64-linux = {
          target = "aarch64-unknown-linux-musl";
          hash = "sha256-tDWYlrtUjgL91y6gyzOV/oqI0g06SkIcRIHlBPjokn8=";
        };
        aarch64-darwin = {
          target = "aarch64-apple-darwin";
          hash = "sha256-vL+nZlC2xYFQWqUXjB55nTf/EvxDo1/xbJC5f6dX5j8=";
        };
      };

      antigravityVersion = "1.0.10";
      antigravityArtifacts = {
        x86_64-linux = {
          path = "linux-x64/cli_linux_x64.tar.gz";
          hash = "sha256-ZUfPmjcifyYAT6S4BUGLHflvVMV7lyPKfRCGTSYQuw8=";
        };
        aarch64-linux = {
          path = "linux-arm/cli_linux_arm64.tar.gz";
          hash = "sha256-RnT6vDaBIh5UyQ0VB3yal6JepxIiAB2r5EvxV26IhZM=";
        };
        aarch64-darwin = {
          path = "darwin-arm/cli_mac_arm64.tar.gz";
          hash = "sha256-yFe1/HA1RgNZ6OZK7kB2jm9SKDWLQnG8fe0Gw+a80mA=";
        };
      };

      claudeCodeVersion = "2.1.215";
      claudeCodeArtifacts = {
        x86_64-linux = {
          platform = "linux-x64";
          hash = "sha256-we//qvNwqhh8tqCd2T1OURxkaJmwB4R2+DeRtmS95/4=";
        };
        aarch64-linux = {
          platform = "linux-arm64";
          hash = "sha256-K0Oj1bB4chfl1zgfrULHMUKSVG/p2564ubN53pBQmzA=";
        };
        aarch64-darwin = {
          platform = "darwin-arm64";
          hash = "sha256-kGCLXFq1BOludzZc6mID0EbikdWbK7Qs8o3LLM353Vg=";
        };
      };
      claudeCodeUrl =
        platform:
        "https://downloads.claude.ai/claude-code-releases/${claudeCodeVersion}/${platform}/claude";

      # Shared by the Nix package's wrapper and the generated PKGBUILD's package(), so the
      # "disable Claude's own updater/health-check" behavior only needs to be declared once.
      claudeCodeWrapperEnv = {
        DISABLE_UPDATES = "1";
        DISABLE_INSTALLATION_CHECKS = "1";
      };
      claudeCodeWrapperExports = lib.concatStrings (
        lib.mapAttrsToList (name: value: "export ${name}=${value}\n") claudeCodeWrapperEnv
      );

      # Renders a pacman PKGBUILD from the same version/url/hash data used to build the Nix
      # package, via Nix string interpolation, instead of hand-writing (and drifting from) a
      # separate copy. `artifacts` is keyed by pacman arch name (x86_64, aarch64), each an
      # { url, sha256Hex } pair.
      mkPkgbuild =
        {
          pname,
          pkgname ? "${pname}-bin",
          version,
          description,
          homepage,
          license ? "LicenseRef-${pname}",
          artifacts,
          packageBody,
        }:
        ''
          # Generated by flake.nix's mkPkgbuild for the '${pname}' package - do not edit by hand.
          # Regenerate with: nix run .#gen-pkgbuilds
          pkgname=${pkgname}
          pkgver=${version}
          pkgrel=1
          pkgdesc="${description}"
          arch=(${lib.concatMapStringsSep " " (arch: "'${arch}'") (lib.attrNames artifacts)})
          url="${homepage}"
          license=('${license}')
          options=('!strip')
          provides=('${pname}')
          conflicts=('${pname}')
        ''
        + lib.concatStrings (
          lib.mapAttrsToList (
            arch: a: ''source_${arch}=("$pkgname-$pkgver-${arch}::${a.url}")'' + "\n"
          ) artifacts
        )
        + lib.concatStrings (
          lib.mapAttrsToList (arch: a: "sha256sums_${arch}=('${a.sha256Hex}')\n") artifacts
        )
        + ''

          package() {
          ${packageBody}
          }
        '';

      claudeCodePkgbuildArtifacts =
        lib.mapAttrs
          (
            arch: system:
            let
              a = claudeCodeArtifacts.${system};
            in
            {
              url = claudeCodeUrl a.platform;
              sha256Hex = builtins.convertHash {
                hash = a.hash;
                toHashFormat = "base16";
                hashAlgo = "sha256";
              };
            }
          )
          {
            x86_64 = "x86_64-linux";
            aarch64 = "aarch64-linux";
          };

      claudeCodePkgbuild = mkPkgbuild {
        pname = "claude-code";
        version = claudeCodeVersion;
        description = "Claude Code CLI";
        homepage = "https://claude.ai/code";
        artifacts = claudeCodePkgbuildArtifacts;
        packageBody = ''
          install -Dm755 "$srcdir/$pkgname-$pkgver-$CARCH" "$pkgdir/opt/$pkgname/bin/claude"
          install -dm755 "$pkgdir/usr/bin"
          cat > "$pkgdir/usr/bin/claude" << 'INNEREOF'
          #!/bin/sh
        ''
        + claudeCodeWrapperExports
        + ''
          exec /opt/$pkgname/bin/claude "$@"
          INNEREOF
            chmod 755 "$pkgdir/usr/bin/claude"'';
      };

      vibeVersion = "2.21.0";
      vibeArtifacts = {
        x86_64-linux = {
          arch = "linux-x86_64";
          hash = "sha256-bn55VqID01SdoA+0yp7nzYtTHnw9qFp37D+bbSvC2bw=";
        };
        aarch64-linux = {
          arch = "linux-aarch64";
          hash = "sha256-NXojcwcjGtp2hfiIHgRaUzTiQoEq8QgMej8YcyF8AQk=";
        };
        aarch64-darwin = {
          arch = "darwin-aarch64";
          hash = "sha256-LguNMpOX2aCF8OCaWBXPm02Fs0M0qjE4Xc4LWG7s/z8=";
        };
      };

      vibeAcpArtifacts = {
        x86_64-linux = {
          arch = "linux-x86_64";
          hash = "sha256-5B3ZQ3U2rmM4P6dk6ktQkG6q3S4ralZR0Bms+VDukro=";
        };
        aarch64-linux = {
          arch = "linux-aarch64";
          hash = "sha256-ubSL6Li33OH+RLUtMMBGaVYRKde0DZxVcPltpNRykbw=";
        };
        aarch64-darwin = {
          arch = "darwin-aarch64";
          hash = "sha256-gTMVZvP8G5MUPLthxDTjHLM6LxYlgoCAu5os5Lmu86A=";
        };
      };

      copilotVersion = "1.0.71";
      copilotArtifacts = {
        x86_64-linux = {
          platform = "linux-x64";
          hash = "sha256-d56bPlI5nY/fW81hd54/HWBnlrpHi2FK1B+4DVIpELs=";
        };
        aarch64-linux = {
          platform = "linux-arm64";
          hash = "sha256-dMx82q7TmPJrfXLH1Buiv/QakDUlrqx3g2HW29igxg0=";
        };
        aarch64-darwin = {
          platform = "darwin-arm64";
          hash = "sha256-DvILAwi24j6dRMFDvwde59Kay7vDhHusuOKWI/LSQ4k=";
        };
      };

      opencodeVersion = "1.18.3";
      opencodeArtifacts = {
        x86_64-linux = {
          platform = "linux-x64";
          hash = "sha256-YPJ7JnnwClEbZTn5fgJEivr1jZxm4kSCheoMUXyoRYM=";
        };
        aarch64-linux = {
          platform = "linux-arm64";
          hash = "sha256-2gpjEXTro4CyodUfnTZPo4EtpDPnJ0PHJHHUtdpZxp0=";
        };
        aarch64-darwin = {
          platform = "darwin-arm64";
          hash = "sha256-/8K3SmfWU56vlbPaN1VvkOhYO0FDOOH4TB4cGpa25hc=";
        };
      };

      protonPassVersion = "1.38.1";
      protonPassArtifacts = {
        x86_64-linux = {
          url = "https://proton.me/download/pass/linux/proton-pass_${protonPassVersion}_amd64.deb";
          hash = "sha512-4jtjW0CVpZNOH4S1NG+gewXDEkjISzHneTdcPOSIeHOQTHbjtMUriCTZpsG1fBlRftiJeHsvF1aMO38VACFsYg==";
        };
        aarch64-darwin = {
          url = "https://proton.me/download/pass/macos/ProtonPass_${protonPassVersion}.dmg";
          hash = "sha512-6TtYT8QtlUTD/FkqT0emQKo4mq/2BR6G8MY+F/Z+Urd8cAX9SGDSrrG/NDgZQtVELCOdQSeZZ4DgFNbM/m6CAw==";
        };
      };

      # Built straight from Proton's own release feed instead of nixpkgs' pkgs.proton-pass,
      # which tracks releases slower than upstream. Mirrors nixpkgs' own proton-pass recipe
      # (pkgs/by-name/pr/proton-pass): unpack the vendor .deb/.dmg and repoint app.asar's
      # process.resourcesPath at the store path since the app is launched via system electron.
      mkProtonPassPackage =
        pkgs: system:
        let
          artifact = protonPassArtifacts.${system};
          src = pkgs.fetchurl { inherit (artifact) url hash; };
          meta = packageMeta // {
            description = "Desktop application for Proton Pass";
            homepage = "https://proton.me/pass";
            mainProgram = "proton-pass";
          };
        in
        if pkgs.stdenv.isDarwin then
          pkgs.stdenv.mkDerivation {
            pname = "proton-pass";
            version = protonPassVersion;
            inherit src meta;

            nativeBuildInputs = [ pkgs.undmg ];
            sourceRoot = ".";

            installPhase = ''
              runHook preInstall
              mkdir -p $out/Applications
              cp -r *.app $out/Applications
              runHook postInstall
            '';
          }
        else
          pkgs.stdenvNoCC.mkDerivation {
            pname = "proton-pass";
            version = protonPassVersion;
            inherit src meta;

            nativeBuildInputs = with pkgs; [
              dpkg
              makeWrapper
              asar
            ];

            dontConfigure = true;
            dontBuild = true;

            # dpkg-deb -x fails on chrome-sandbox's SUID bit inside the build sandbox;
            # tar --no-same-permissions applies the umask instead, dropping it (unused here
            # anyway, since the app runs under the system electron, not the bundled one).
            unpackPhase = ''
              runHook preUnpack
              dpkg-deb --fsys-tarfile "$src" | tar -x --no-same-owner --no-same-permissions
              runHook postUnpack
            '';

            preInstall = ''
              asar extract usr/lib/proton-pass/resources/app.asar tmp
              rm usr/lib/proton-pass/resources/app.asar
              substituteInPlace tmp/.webpack/main/index.js \
                --replace-fail "process.resourcesPath" "'$out/share/proton-pass'"
              asar pack tmp/ usr/lib/proton-pass/resources/app.asar
              rm -fr tmp
            '';

            installPhase = ''
              runHook preInstall
              mkdir -p $out/share/proton-pass
              cp -r usr/share/ $out/
              cp -r usr/lib/proton-pass/resources/{app.asar,assets} $out/share/proton-pass/
              runHook postInstall
            '';

            preFixup = ''
              makeWrapper ${lib.getExe pkgs.electron} $out/bin/proton-pass \
                --add-flags $out/share/proton-pass/app.asar \
                --set-default ELECTRON_FORCE_IS_PACKAGED 1 \
                --set-default ELECTRON_IS_DEV 0 \
                --inherit-argv0
            '';
          };

      claudeDesktopVersion = "1.22209.3";
      claudeDesktopArtifacts = {
        x86_64-linux = {
          arch = "amd64";
          hash = "sha256-1Cf0askjPbxNikQaYC8J91C4pfBdH8egAoXXps4HZVw=";
        };
        aarch64-linux = {
          arch = "arm64";
          hash = "sha256-Vcy0eLItcbRuZpWC565Nb0T8bf8LPVFakWMEnatANLI=";
        };
      };

      # Repackages Anthropic's official .deb (Linux-only beta, no upstream nixpkgs package yet).
      # Keeps the official co-located app tree intact (usr/lib/claude-desktop/{claude-desktop,resources/})
      # so /proc/self/exe and process.resourcesPath resolve correctly inside the store path,
      # instead of running the app under nixpkgs' electron.
      mkClaudeDesktopPackage =
        pkgs: system:
        let
          artifact = claudeDesktopArtifacts.${system};
        in
        pkgs.stdenv.mkDerivation {
          pname = "claude-desktop";
          version = claudeDesktopVersion;

          src = pkgs.fetchurl {
            url = "https://downloads.claude.ai/claude-desktop/apt/stable/pool/main/c/claude-desktop/claude-desktop_${claudeDesktopVersion}_${artifact.arch}.deb";
            inherit (artifact) hash;
          };

          nativeBuildInputs = with pkgs; [
            dpkg
            autoPatchelfHook
            makeWrapper
          ];

          buildInputs = with pkgs; [
            alsa-lib
            at-spi2-atk
            at-spi2-core
            atk
            cairo
            cups
            dbus
            expat
            glib
            gtk3
            libcap_ng
            libgbm
            libseccomp
            libxkbcommon
            nspr
            nss
            pango
            stdenv.cc.cc.lib
            systemd
            libx11
            libxcomposite
            libxdamage
            libxext
            libxfixes
            libxrandr
            libxcb
          ];

          # dlopen'd at runtime rather than DT_NEEDED, so autoPatchelf won't find them on its own.
          runtimeDependencies = map lib.getLib (
            with pkgs;
            [
              libGL
              libayatana-appindicator
              libnotify
              libpulseaudio
              libsecret
              libuuid
              pciutils
              pipewire
              systemd
              wayland
              libxtst
            ]
          );

          # dpkg-deb -x fails on chrome-sandbox's SUID bit inside the build sandbox;
          # tar --no-same-permissions applies the umask instead, dropping it (the store
          # couldn't represent it anyway - relies on unprivileged userns, same as nixpkgs' slack/signal-desktop).
          unpackPhase = ''
            runHook preUnpack
            dpkg-deb --fsys-tarfile "$src" | tar -x --no-same-owner --no-same-permissions
            runHook postUnpack
          '';

          dontConfigure = true;
          dontBuild = true;
          dontStrip = true;

          installPhase = ''
            runHook preInstall

            mkdir -p $out
            cp -a usr/lib usr/share $out/
            makeWrapper $out/lib/claude-desktop/claude-desktop \
              $out/bin/claude-desktop \
              --prefix VK_ADD_DRIVER_FILES : \
                "${pkgs.addDriverRunpath.driverLink}/share/vulkan/icd.d"

            runHook postInstall
          '';

          preFixup = ''
            addAutoPatchelfSearchPath "$out/lib/claude-desktop"
          '';

          # The bundled ANGLE libs dlopen the glvnd EGL dispatcher by soname and only
          # carry their own DT_NEEDED on their runpath, so libGL must be added explicitly.
          appendRunpaths = [
            "${lib.getLib pkgs.libGL}/lib"
            "${pkgs.addDriverRunpath.driverLink}/lib"
          ];

          meta = packageMeta // {
            description = "Claude Desktop for Linux (repackaged official .deb)";
            homepage = "https://claude.ai";
            mainProgram = "claude-desktop";
          };
        };

      kimiVersion = "1.49.0";
      kimiArtifacts = {
        x86_64-linux = {
          platform = "x86_64-unknown-linux-gnu";
          hash = "sha256-bOC4P1g8RaZMyfUf/n4ajgPueazaaZRfz4wjNBudiS8=";
        };
        aarch64-linux = {
          platform = "aarch64-unknown-linux-gnu";
          hash = "sha256-WsVMq84W7eJ7nSBpubiO3uJVKGRue7W++pmAocpx/rs=";
        };
        aarch64-darwin = {
          platform = "aarch64-apple-darwin";
          hash = "sha256-FQGLILIDruCWWP3GSEDEhG/BfBCNjboaGalVgdPOKSE=";
        };
      };
    in
    {
      packages = forEachSupportedSystem (
        { pkgs, system }:
        let
          grokArtifact = grokArtifacts.${system};
          codexArtifact = codexArtifacts.${system};
          antigravityArtifact = antigravityArtifacts.${system};
          claudeCodeArtifact = claudeCodeArtifacts.${system};
          copilotArtifact = copilotArtifacts.${system};
          opencodeArtifact = opencodeArtifacts.${system};
          kimiArtifact = kimiArtifacts.${system};

          grok = mkBinaryPackage pkgs {
            pname = "grok-cli";
            version = grokVersion;
            src = pkgs.fetchurl {
              url = "https://x.ai/cli/grok-${grokVersion}-${grokArtifact.platform}";
              inherit (grokArtifact) hash;
            };
            executableName = "grok";
            extraExecutables = [ "agent" ];
            meta = {
              description = "Grok CLI";
              homepage = "https://x.ai/cli";
              mainProgram = "grok";
            };
          };

          codex = mkTarballPackage pkgs {
            pname = "codex-cli";
            version = codexVersion;
            src = pkgs.fetchurl {
              url = "https://github.com/openai/codex/releases/download/rust-v${codexVersion}/codex-package-${codexArtifact.target}.tar.gz";
              inherit (codexArtifact) hash;
            };
            binaryPath = "bin/codex";
            executableName = "codex";
            meta = {
              description = "OpenAI Codex CLI";
              homepage = "https://github.com/openai/codex";
              mainProgram = "codex";
            };
          };

          antigravity = mkTarballPackage pkgs {
            pname = "antigravity-cli";
            version = antigravityVersion;
            src = pkgs.fetchurl {
              url = "https://storage.googleapis.com/antigravity-public/antigravity-cli/${antigravityVersion}-6349723456634880/${antigravityArtifact.path}";
              inherit (antigravityArtifact) hash;
            };
            binaryPath = "antigravity";
            executableName = "agy";
            meta = {
              description = "Google Antigravity CLI";
              homepage = "https://antigravity.google/cli";
              mainProgram = "agy";
            };
          };

          claude-code = mkBinaryPackage pkgs {
            pname = "claude-code";
            version = claudeCodeVersion;
            src = pkgs.fetchurl {
              url = "https://downloads.claude.ai/claude-code-releases/${claudeCodeVersion}/${claudeCodeArtifact.platform}/claude";
              inherit (claudeCodeArtifact) hash;
            };
            executableName = "claude";
            postInstall = ''
              mv $out/bin/claude $out/bin/.claude-unwrapped
              cat > $out/bin/claude <<'EOF'
              #!${pkgs.runtimeShell}
            ''
            + claudeCodeWrapperExports
            + ''
              exec "$(dirname "$0")/.claude-unwrapped" "$@"
              EOF
              chmod 755 $out/bin/claude
            '';
            meta = {
              description = "Claude Code CLI";
              homepage = "https://claude.ai/code";
              mainProgram = "claude";
            };
          };

          vibeArtifact = vibeArtifacts.${system};
          vibeAcpArtifact = vibeAcpArtifacts.${system};

          vibe = mkZipPackage pkgs {
            pname = "mistral-vibe";
            version = vibeVersion;
            url = "https://github.com/mistralai/mistral-vibe/releases/download/v${vibeVersion}/vibe-${vibeArtifact.arch}-${vibeVersion}.zip";
            inherit (vibeArtifact) hash;
            binaryPath = "vibe";
            executableName = "vibe";
            meta = {
              description = "Mistral Vibe CLI";
              homepage = "https://github.com/mistralai/mistral-vibe";
              mainProgram = "vibe";
            };
          };

          vibe-acp = mkZipPackage pkgs {
            pname = "mistral-vibe-acp";
            version = vibeVersion;
            url = "https://github.com/mistralai/mistral-vibe/releases/download/v${vibeVersion}/vibe-acp-${vibeAcpArtifact.arch}-${vibeVersion}.zip";
            inherit (vibeAcpArtifact) hash;
            binaryPath = "vibe-acp";
            executableName = "vibe-acp";
            meta = {
              description = "Mistral Vibe ACP";
              homepage = "https://github.com/mistralai/mistral-vibe";
              mainProgram = "vibe-acp";
            };
          };

          copilot = mkTarballPackage pkgs {
            pname = "copilot-cli";
            version = copilotVersion;
            src = pkgs.fetchurl {
              url = "https://github.com/github/copilot-cli/releases/download/v${copilotVersion}/copilot-${copilotArtifact.platform}.tar.gz";
              inherit (copilotArtifact) hash;
            };
            binaryPath = "copilot";
            executableName = "copilot";
            meta = {
              description = "GitHub Copilot CLI";
              homepage = "https://github.com/github/copilot-cli";
              mainProgram = "copilot";
            };
          };

          opencode =
            let
              ext = if lib.strings.hasSuffix "darwin" system then "zip" else "tar.gz";
              opencodeUrl = "https://github.com/anomalyco/opencode/releases/download/v${opencodeVersion}/opencode-${opencodeArtifact.platform}.${ext}";
            in
            if ext == "zip" then
              mkZipPackage pkgs {
                pname = "opencode";
                version = opencodeVersion;
                url = opencodeUrl;
                inherit (opencodeArtifact) hash;
                binaryPath = "opencode";
                meta = {
                  description = "opencode CLI";
                  homepage = "https://github.com/anomalyco/opencode";
                  mainProgram = "opencode";
                };
              }
            else
              mkTarballPackage pkgs {
                pname = "opencode";
                version = opencodeVersion;
                src = pkgs.fetchurl {
                  url = opencodeUrl;
                  inherit (opencodeArtifact) hash;
                };
                binaryPath = "opencode";
                meta = {
                  description = "opencode CLI";
                  homepage = "https://github.com/anomalyco/opencode";
                  mainProgram = "opencode";
                };
              };

          kimi = mkTarballPackage pkgs {
            pname = "kimi-cli";
            version = kimiVersion;
            src = pkgs.fetchurl {
              url = "https://github.com/MoonshotAI/kimi-cli/releases/download/${kimiVersion}/kimi-${kimiVersion}-${kimiArtifact.platform}.tar.gz";
              inherit (kimiArtifact) hash;
            };
            binaryPath = "kimi";
            executableName = "kimi";
            meta = {
              description = "Kimi (Moonshot AI) CLI";
              homepage = "https://github.com/MoonshotAI/kimi-cli";
              mainProgram = "kimi";
            };
          };
        in
        {
          inherit
            antigravity
            codex
            copilot
            grok
            kimi
            opencode
            vibe
            vibe-acp
            ;
          inherit claude-code;

          agy = antigravity;
          claude = claude-code;
          default = codex;
        }
        // lib.optionalAttrs (claudeDesktopArtifacts ? ${system}) {
          claude-desktop = mkClaudeDesktopPackage pkgs system;
        }
        // lib.optionalAttrs (protonPassArtifacts ? ${system}) {
          proton-pass = mkProtonPassPackage pkgs system;
        }
        // lib.optionalAttrs (lib.strings.hasSuffix "linux" system) {
          claude-code-pkgbuild = pkgs.writeText "PKGBUILD" claudeCodePkgbuild;
        }
      );

      devShells = forEachSupportedSystem (
        { pkgs, system }:
        {
          default = pkgs.mkShellNoCC {
            packages = with pkgs; [
              self.formatter.${system}
            ];
          };
        }
      );

      # Copies the generated PKGBUILD(s) from the Nix store into ./pkgbuilds/<name>/PKGBUILD
      # in the working tree, so they can be reviewed and committed. Run with `nix run .#gen-pkgbuilds`.
      apps = forEachSupportedSystem (
        { pkgs, system }:
        lib.optionalAttrs (lib.strings.hasSuffix "linux" system) {
          gen-pkgbuilds = {
            type = "app";
            meta.description = "Write generated PKGBUILDs into ./pkgbuilds/";
            program = toString (
              pkgs.writeShellScript "gen-pkgbuilds" ''
                set -euo pipefail
                out="$PWD/pkgbuilds/claude-code-bin"
                mkdir -p "$out"
                install -m644 ${pkgs.writeText "PKGBUILD" claudeCodePkgbuild} "$out/PKGBUILD"
                echo "wrote $out/PKGBUILD"
              ''
            );
          };
        }
      );

      formatter = forEachSupportedSystem ({ pkgs, ... }: pkgs.nixfmt);
    };
}
