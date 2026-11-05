{ pkgs, throne }:
let
  inherit (pkgs) lib;
  wrapper = pkgs.pkgsStatic.callPackage (pkgs.path + "/nixos/modules/security/wrappers/wrapper.nix") {
    sourceProg = "${throne}/share/throne/ThroneCore";
    unsecvars = lib.overrideDerivation (pkgs.srcOnly pkgs.glibc) (
      { name, ... }:
      {
        name = "${name}-unsecvars";
        installPhase = ''
          mkdir $out
          cp sysdeps/generic/unsecvars.h $out
        '';
      }
    );
  };
in
pkgs.writeShellApplication {
  name = "throne-tun-setup";
  runtimeInputs = with pkgs; [
    coreutils
    getent
    shadow
    libcap
  ];
  runtimeEnv = {
    THRONE_PACKAGE = toString throne;
    THRONE_WRAPPER = "${wrapper}/bin/security-wrapper";
    THRONE_POLKIT_RULE = toString (
      pkgs.writeText "throne-tun.rules" (builtins.readFile ./throne-tun.rules)
    );
  };
  text = builtins.readFile ../scripts/setup-throne-tun.sh;
}
