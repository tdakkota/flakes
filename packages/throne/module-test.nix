{
  self,
  nixpkgs,
  pkgs,
  system,
}:
let
  machine = nixpkgs.lib.nixosSystem {
    inherit system;
    modules = [
      self.nixosModules.throne
      {
        boot.isContainer = true;
        networking.useHostResolvConf = false;
        system.stateVersion = "26.05";
        services.resolved.enable = true;
        users.users.test.isNormalUser = true;
        programs.throne = {
          enable = true;
          tunMode = {
            enable = true;
            allowedUsers = [ "test" ];
          };
        };
      }
    ];
  };
  cfg = machine.config;
  wrapper = cfg.security.wrappers.ThroneCore;
in
assert builtins.all (entry: entry.assertion) cfg.assertions;
assert wrapper.source == "${self.packages.${system}.throne}/share/throne/ThroneCore";
assert wrapper.group == "throne-tun";
assert wrapper.permissions == "u+rx,g+rx";
assert wrapper.capabilities == "cap_net_admin,cap_net_raw,cap_net_bind_service+ep";
assert !wrapper.setuid;
assert !cfg.security.wrappers.throne-core.enable;
assert cfg.users.groups.throne-tun.members == [ "test" ];
pkgs.runCommand "throne-module-check" { } ''
  touch "$out"
''
