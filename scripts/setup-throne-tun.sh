if (( EUID != 0 )); then
  echo "Run this optional installer with sudo after reviewing it." >&2
  exit 1
fi
if [[ $# != 2 || $1 != --user || ! $2 =~ ^[a-zA-Z_][a-zA-Z0-9_-]*\$?$ ]]; then
  echo "Usage: throne-tun-setup --user USER" >&2
  exit 2
fi
target_user=$2
getent passwd "$target_user" >/dev/null
if [[ ! -d /nix/var/nix/gcroots ]]; then
  echo "A multi-user Nix installation is required." >&2
  exit 1
fi
for protected_path in "$THRONE_PACKAGE" "$THRONE_WRAPPER" /nix/var/nix/gcroots; do
  permissions=$(stat -Lc %a "$protected_path")
  if [[ $(stat -Lc %u "$protected_path") != 0 ]] || (( (8#$permissions & 0022) != 0 )); then
    echo "Refusing an untrusted Nix package or GC root: $protected_path" >&2
    exit 1
  fi
done

helper_dir=/usr/local/libexec/throne
gc_root=/nix/var/nix/gcroots/throne-tun
for directory in /usr/local /usr/local/libexec "$helper_dir"; do
  if [[ -e "$directory" ]]; then
    if [[ -L "$directory" || $(stat -c %u "$directory") != 0 ]]; then
      echo "Refusing a non-root-owned or symlinked helper directory: $directory" >&2
      exit 1
    fi
    permissions=$(stat -c %a "$directory")
    if (( (8#$permissions & 0022) != 0 )); then
      echo "Refusing a group/other-writable helper directory: $directory" >&2
      exit 1
    fi
  fi
done
if ! getent group throne-tun >/dev/null; then
  groupadd --system throne-tun
fi
usermod -a -G throne-tun "$target_user"
install -d -o root -g root -m 0755 "$helper_dir"
temporary=$(mktemp "$helper_dir/.ThroneCore.XXXXXX")
trap 'rm -f "$temporary"' EXIT
install -o root -g throne-tun -m 0000 "$THRONE_WRAPPER" "$temporary"
setcap cap_setpcap,cap_net_admin,cap_net_raw,cap_net_bind_service+ep "$temporary"
chmod 0750 "$temporary"
mv -fT "$temporary" "$helper_dir/ThroneCore"
ln -sfnT "$THRONE_PACKAGE" "$gc_root"
if [[ ! -d /etc/polkit-1/rules.d ]]; then
  install -d -o root -g root -m 0755 /etc/polkit-1/rules.d
fi
install -o root -g root -m 0644 "$THRONE_POLKIT_RULE" /etc/polkit-1/rules.d/60-throne-tun.rules
echo "Installed a root-owned capability wrapper; no Nix store file was modified."
echo "Use a fresh 'newgrp throne-tun' shell as $target_user, or reboot to refresh user-service groups."
echo "Then start Throne and enable TUN explicitly."
echo "Repeat this installer after upgrading Throne to retarget the pinned core."
