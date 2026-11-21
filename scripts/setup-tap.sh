#!/usr/bin/env bash
# Recreate the host side of the VM network after a reboot.
#
#   br0  = 10.200.0.1/24, the gateway every VM uses (set in boot_args)
#   tapN = one tap per VM slot, plugged into br0 like a switch port
#          slot N -> tapN -> guest IP 10.200.0.(N+2)
#
# Usage: sudo ./scripts/setup-tap.sh [number-of-taps]   (default 2)
# The number must match the pool size in server.go (network.NewPool).
# Safe to run more than once.
set -euo pipefail

BRIDGE=br0
GATEWAY=10.200.0.1/24
COUNT=${1:-2}
OWNER=${SUDO_USER:-$USER}   # Firecracker runs as this user, so it must own the taps

if [[ $EUID -ne 0 ]]; then
  echo "run with sudo: sudo $0 [count]" >&2
  exit 1
fi

# 1. The bridge (virtual switch). The gateway IP lives here, not on a tap.
if ! ip link show "$BRIDGE" &>/dev/null; then
  ip link add "$BRIDGE" type bridge
  echo "created $BRIDGE"
fi
# "replace" instead of "add" so a second run does not fail with "File exists".
ip addr replace "$GATEWAY" dev "$BRIDGE"
ip link set "$BRIDGE" up

# 2. One tap per slot, each plugged into the bridge.
for ((i = 0; i < COUNT; i++)); do
  tap="tap$i"
  if ! ip link show "$tap" &>/dev/null; then
    ip tuntap add dev "$tap" mode tap user "$OWNER"
    echo "created $tap (owner $OWNER)"
  fi
  # Old single-tap setups put the gateway IP on tap0; a bridge port must not have it.
  ip addr flush dev "$tap" scope global
  ip link set "$tap" master "$BRIDGE"
  ip link set "$tap" up
done

# 3. Show the result. Taps say NO-CARRIER until a VM opens them; that is normal.
ip -br addr show "$BRIDGE"
/usr/sbin/bridge link | grep "master $BRIDGE" || true
