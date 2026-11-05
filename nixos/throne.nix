{ packages }:
{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.programs.throne;
in
{
  options.programs.throne.tunMode.allowedUsers = lib.mkOption {
    type = lib.types.listOf lib.types.str;
    default = [ ];
    description = "Users allowed to run ThroneCore with network capabilities.";
  };
  config = lib.mkIf cfg.enable {
    programs.throne.package = lib.mkDefault packages.${pkgs.stdenv.hostPlatform.system}.throne;
    assertions = [
      {
        assertion = !cfg.tunMode.enable || cfg.tunMode.allowedUsers != [ ];
        message = "Set programs.throne.tunMode.allowedUsers when enabling TUN.";
      }
    ];
    # Older Nixpkgs names the core wrapper differently.
    security.wrappers.throne-core.enable = lib.mkForce false;
    security.wrappers.ThroneCore = lib.mkIf cfg.tunMode.enable {
      source = lib.mkForce "${cfg.package}/share/throne/ThroneCore";
      owner = lib.mkForce "root";
      group = lib.mkForce "throne-tun";
      permissions = lib.mkForce "u+rx,g+rx";
      setuid = lib.mkForce cfg.tunMode.setuid;
      capabilities = lib.mkForce (
        if cfg.tunMode.setuid then "" else "cap_net_admin,cap_net_raw,cap_net_bind_service+ep"
      );
    };
    users.groups.throne-tun = lib.mkIf cfg.tunMode.enable {
      members = cfg.tunMode.allowedUsers;
    };
    security.polkit.enable = lib.mkIf cfg.tunMode.enable true;
    security.polkit.extraConfig = lib.mkIf (cfg.tunMode.enable && config.services.resolved.enable) (
      builtins.readFile ../nix/throne-tun.rules
    );
  };
}
