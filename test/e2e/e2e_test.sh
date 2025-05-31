#!/usr/bin/env bash

set -euo pipefail
# set -x

TEMP_TEMP_DIR=/tmp/tap-etherip
ENV_BASE_PATH="$TEMP_TEMP_DIR"

rm -rf "$TEMP_TEMP_DIR"

mkdir -p "$ENV_BASE_PATH"
SERVICE_SRC="$(dirname "$0")/tap-etherip@.service"
SERVICE_DST="/etc/systemd/system/tap-etherip@.service"

cp "$SERVICE_SRC" "$SERVICE_DST"
systemctl daemon-reload

MOCK_BIN_SRC="$(dirname "$0")/sleep-infinity.sh"
MOCK_BIN_DST="$TEMP_TEMP_DIR"
cp "$MOCK_BIN_SRC" "$MOCK_BIN_DST"

CONFIG_DIR=./testdata/configs
CONFIGS=("$CONFIG_DIR"/*.json)

test_ns="tunnel1"

cleanup() {
  pkill -f "sleep-infinity.sh" || true
#   ip netns del "$test_ns" 2>/dev/null || true
#   rm -rf "$TEMP_TEMP_DIR"
}
trap cleanup EXIT

if ! ip netns list | grep -q "$test_ns"; then
  echo "Creating network namespace: $test_ns"
  ip netns add "$test_ns"
fi

for CONFIG in "${CONFIGS[@]}"; do
  echo "Testing with config: $CONFIG"
  ./tap-etherip-config-sync --log-level debug --config "$CONFIG" --env-base-path "$ENV_BASE_PATH"
  sleep 1
  if pgrep -f "sleep-infinity.sh" > /dev/null; then
    echo "Process started as expected for $CONFIG"
  else
    echo "Process did not start as expected for $CONFIG"
    exit 1
  fi

  # 出力するのみ
  ps -ef | grep "sleep-infinity.sh" | grep -v grep
done
