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

      grokVersion = "0.2.59";
      grokArtifacts = {
        x86_64-linux = {
          platform = "linux-x86_64";
          hash = "sha256-G0Fh0vr0X8yaNJBjHM/jurNcDROAeiQ9g16wCz9vIik=";
        };
        aarch64-linux = {
          platform = "linux-aarch64";
          hash = "sha256-BdzruY/U4mbV5YqvO5jKEqZxtxTVMg3MNnFiINi0HKg=";
        };
        aarch64-darwin = {
          platform = "macos-aarch64";
          hash = "sha256-cUkzP+npR+vXR6iIYoMJZXtqZS/6ZclJsBCbsold/pg=";
        };
      };

      codexVersion = "0.141.0";
      codexArtifacts = {
        x86_64-linux = {
          target = "x86_64-unknown-linux-musl";
          hash = "sha256-CRyKLic3DEFAf6HLZH/pBb1P1w5GicE+/+4KLc4bKwc=";
        };
        aarch64-linux = {
          target = "aarch64-unknown-linux-musl";
          hash = "sha256-twAwM4WS3j42Hzzeg9Yk+IBh3zAKvjG2IHWlxaBYpvw=";
        };
        aarch64-darwin = {
          target = "aarch64-apple-darwin";
          hash = "sha256-o38WiPabOLHO0FYIbSM+ZHhHQP4ZoqFeOtwcaCjhuTM=";
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

      claudeCodeVersion = "2.1.185";
      claudeCodeArtifacts = {
        x86_64-linux = {
          platform = "linux-x64";
          hash = "sha256-4SRjOGmfBO4OYn3uP21O16C6tI4FFL3mnG2tQ7wwOVI=";
        };
        aarch64-linux = {
          platform = "linux-arm64";
          hash = "sha256-24gIEiclBEVd9zFg2S+t+TcO2mhMIZzr+OYrCiYssvg=";
        };
        aarch64-darwin = {
          platform = "darwin-arm64";
          hash = "sha256-ooDCOyEFJSGPW9hvABydvIm54HQQF1xak1UES/rcCvE=";
        };
      };

      vibeVersion = "2.17.1";
      vibeArtifacts = {
        x86_64-linux = {
          arch = "linux-x86_64";
          hash = "sha256-N9uGYHSFfxmiznMdJPzy5bRVrVUXt3R3+y16u0NzSGk=";
        };
        aarch64-linux = {
          arch = "linux-aarch64";
          hash = "sha256-MFHNvRHpsCt6NzADuluy4hUhMUFs6rzfaTztomgSqqo=";
        };
        aarch64-darwin = {
          arch = "darwin-aarch64";
          hash = "sha256-fSp/gZ6lCekwTQDPGAlgTbThXoXuwBq0QEsS2wa+xzI=";
        };
      };

      vibeAcpArtifacts = {
        x86_64-linux = {
          arch = "linux-x86_64";
          hash = "sha256-TqxjAnzlXLCGIWvMrETBIprSDpOOwATfUmGm3+nkjqg=";
        };
        aarch64-linux = {
          arch = "linux-aarch64";
          hash = "sha256-bGyvD+nCTvVwc8VAw04SiZJV3N04EFFNtfL+QqaZx10=";
        };
        aarch64-darwin = {
          arch = "darwin-aarch64";
          hash = "sha256-6W3wSe+v0eCkOlsyk2jBthJHTeaWuFf5M/J9U+0cRI8=";
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
              export DISABLE_UPDATES=1
              export DISABLE_INSTALLATION_CHECKS=1
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
        in
        {
          inherit
            antigravity
            codex
            grok
            vibe
            vibe-acp
            ;
          inherit claude-code;

          agy = antigravity;
          claude = claude-code;
          default = codex;
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

      formatter = forEachSupportedSystem ({ pkgs, ... }: pkgs.nixfmt);
    };
}
