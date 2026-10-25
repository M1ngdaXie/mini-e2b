#!/usr/bin/env bash
# Recreate the host side of the VM network after a reboot.
# tap0 = 10.200.0.1/24 on the host; the guest uses 10.200.0.2 (set in boot_args).
# Safe to run more than once.
set -euo pipefail

TAP=tap0
HOST_IP=10.200.0.1/24
OWNER=${SUDO_USER:-$USER}   # FC runs as this user, so it must own the tap

if [[ $EUID -ne 0 ]]; then
  echo "run with sudo: sudo $0" >&2
  exit 1
fi

if ! ip link show "$TAP" &>/dev/null; then
  ip tuntap add dev "$TAP" mode tap user "$OWNER"
  echo "created $TAP (owner $OWNER)"
fi

# "replace" instead of "add" so a second run does not fail with "File exists".
ip addr replace "$HOST_IP" dev "$TAP"
ip link set "$TAP" up

ip -br addr show "$TAP"
