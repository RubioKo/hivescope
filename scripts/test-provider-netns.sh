#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only
# From the repository root on Linux, run: bash scripts/test-provider-netns.sh
# Requires Go, cached modules, iproute2 and sudo access to create namespaces.
set -euo pipefail

work=$(mktemp -d)
client="hs-client-$$"
provider_a="hs-a-$$"
provider_b="hs-b-$$"
namespaces=()
server_pid=
cleanup() {
  if [[ -n "$server_pid" ]]; then
    wait "$server_pid" || true  # the test binary has a 25-second timeout
    cat "$work/provider.log"
  fi
  for namespace in "${namespaces[@]}"; do
    sudo ip netns delete "$namespace"
  done
  rm -f -- "$work/conformance.test" "$work/provider.log"
  rmdir -- "$work"
}
trap cleanup EXIT

GOPROXY=off go test -mod=readonly -c -o "$work/conformance.test" ./internal/conformance
for namespace in "$client" "$provider_a" "$provider_b"; do
  sudo ip netns add "$namespace"
  namespaces+=("$namespace")
  sudo ip -n "$namespace" link set lo up
done

# The veth links and routes exist only inside these three namespaces.
# Both providers own the exact IPv4 address from the published HSRP.
link_provider() {
  local namespace=$1 interface=$2 client_ip=$3 provider_ip=$4
  sudo ip -n "$client" link add "$interface" type veth peer name peer0 netns "$namespace"
  sudo ip -n "$client" addr add "$client_ip/30" dev "$interface"
  sudo ip -n "$client" link set "$interface" up
  sudo ip -n "$namespace" addr add "$provider_ip/30" dev peer0
  sudo ip -n "$namespace" link set peer0 up
  sudo ip -n "$namespace" addr add 192.0.2.10/32 dev lo
}
link_provider "$provider_a" to_a 198.18.0.1 198.18.0.2
link_provider "$provider_b" to_b 198.18.0.5 198.18.0.6

uid=$(id -u)
run_role() {
  local namespace=$1 role=$2 host=$3 pin=$4
  sudo ip netns exec "$namespace" sudo -u "#$uid" -- env \
    HIVESCOPE_NETNS_ROLE="$role" HIVESCOPE_NETNS_HOST="$host" HIVESCOPE_NETNS_PIN="$pin" \
    "$work/conformance.test" -test.run '^TestCatalogPinAcrossNetworkHosts$' -test.v -test.timeout 25s
}
run_case() {
  local namespace=$1 gateway=$2 host=$3 pin=$4
  sudo ip -n "$client" route replace 192.0.2.10/32 via "$gateway"
  : >"$work/provider.log"
  run_role "$namespace" provider "$host" "$pin" >>"$work/provider.log" 2>&1 &
  server_pid=$!
  # Wait for the listener's readiness line, without opening a probe connection.
  until grep -Fxq 'HiveScope namespace provider ready' "$work/provider.log"; do
    kill -0 "$server_pid" 2>/dev/null || return 1
    sleep 0.05
  done
  run_role "$client" requester "$host" "$pin"
  wait "$server_pid"
  server_pid=
  cat "$work/provider.log"
}
run_case "$provider_a" 198.18.0.2 A A
run_case "$provider_b" 198.18.0.6 B A
run_case "$provider_b" 198.18.0.6 B B
