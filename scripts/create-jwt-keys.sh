#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KEYS_DIR="${KEYS_DIR:-$ROOT_DIR/keys}"

mkdir -p "$KEYS_DIR"

openssl genrsa -out "$KEYS_DIR/private_access.pem" 2048
openssl rsa -in "$KEYS_DIR/private_access.pem" -pubout -out "$KEYS_DIR/public_access.pem"

openssl genrsa -out "$KEYS_DIR/private_refresh.pem" 2048
openssl rsa -in "$KEYS_DIR/private_refresh.pem" -pubout -out "$KEYS_DIR/public_refresh.pem"

chmod 600 "$KEYS_DIR/private_access.pem" "$KEYS_DIR/private_refresh.pem"
chmod 644 "$KEYS_DIR/public_access.pem" "$KEYS_DIR/public_refresh.pem"

echo "Generated JWT keys in $KEYS_DIR"
