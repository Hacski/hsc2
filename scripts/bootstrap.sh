#!/usr/bin/env sh
set -eu

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
DATA_DIR="${HSC2_DATA_DIR:-./data}"

mkdir -p "$DATA_DIR/db" "$DATA_DIR/certs"

if [ ! -f "$DATA_DIR/.install_key" ]; then
    dd if=/dev/urandom bs=32 count=1 2>/dev/null | base64 > "$DATA_DIR/.install_key"
    chmod 600 "$DATA_DIR/.install_key"
fi

export HSC2_INSTALL_KEY
HSC2_INSTALL_KEY="$(cat "$DATA_DIR/.install_key")"

docker compose -f "$COMPOSE_FILE" pull --quiet 2>/dev/null || true
docker compose -f "$COMPOSE_FILE" up -d --build

echo "hsc2 team server is up"
echo "DB path:   $DATA_DIR/db"
echo "Cert path: $DATA_DIR/certs"
