#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"

if ! command -v openssl >/dev/null 2>&1; then
	echo "openssl is required. On Ubuntu/WSL install it with: sudo apt update && sudo apt install -y openssl" >&2
	exit 1
fi

if [[ -f "$ENV_FILE" ]]; then
	set -a
	# shellcheck disable=SC1090
	. "$ENV_FILE"
	set +a
fi

CERTS_DIR="${CERTS_DIR:-$ROOT_DIR/certs}"
PRIVATE_KEY_PATH="${GRPC_TLS_KEY_PATH:-$CERTS_DIR/private_key.pem}"
CERTIFICATE_PATH="${GRPC_TLS_CERT_PATH:-$CERTS_DIR/public_key.pem}"
PRIVATE_KEY_PATH="${PRIVATE_KEY_PATH/#\/app/$ROOT_DIR}"
CERTIFICATE_PATH="${CERTIFICATE_PATH/#\/app/$ROOT_DIR}"
CA_PRIVATE_KEY_PATH="$CERTS_DIR/local_ca_private_key.pem"
CA_CERTIFICATE_PEM_PATH="$CERTS_DIR/local_ca_cert.pem"
CA_CERTIFICATE_CRT_PATH="$CERTS_DIR/local_ca_cert.crt"
VALID_DAYS="${TLS_CERT_DAYS:-365}"
CA_VALID_DAYS="${TLS_CA_CERT_DAYS:-3650}"

if [[ "${FORCE:-0}" != "1" ]]; then
	if [[ -e "$PRIVATE_KEY_PATH" || -e "$CERTIFICATE_PATH" || -e "$CA_PRIVATE_KEY_PATH" || -e "$CA_CERTIFICATE_PEM_PATH" || -e "$CA_CERTIFICATE_CRT_PATH" ]]; then
		echo "TLS files already exist in $CERTS_DIR. Remove them first or rerun with FORCE=1." >&2
		exit 1
	fi
fi

mkdir -p "$CERTS_DIR" "$(dirname "$PRIVATE_KEY_PATH")" "$(dirname "$CERTIFICATE_PATH")"

TMP_CONFIG="$(mktemp)"
SERVER_CSR_PATH="$(mktemp)"
CA_SERIAL_PATH="$(mktemp)"
rm -f "$CA_SERIAL_PATH"
cleanup() {
	rm -f "$TMP_CONFIG" "$SERVER_CSR_PATH" "$CA_SERIAL_PATH"
}
trap cleanup EXIT

cat > "$TMP_CONFIG" <<'EOF'
[server_req_ext]
basicConstraints = critical, CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
subjectKeyIdentifier = hash

[server_cert_ext]
basicConstraints = critical, CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid,issuer

[ca_ext]
basicConstraints = critical, CA:TRUE, pathlen:0
keyUsage = critical, keyCertSign, cRLSign
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always,issuer

[alt_names]
DNS.1 = localhost
DNS.2 = neuraclinic-auth
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

openssl req \
	-x509 \
	-nodes \
	-newkey rsa:2048 \
	-sha256 \
	-days "$CA_VALID_DAYS" \
	-keyout "$CA_PRIVATE_KEY_PATH" \
	-out "$CA_CERTIFICATE_PEM_PATH" \
	-subj "/C=MX/ST=Sonora/L=Hermosillo/O=Neuraclinic/OU=Development/CN=Neuraclinic Auth Local Dev CA" \
	-config "$TMP_CONFIG" \
	-extensions ca_ext

openssl req \
	-new \
	-nodes \
	-newkey rsa:2048 \
	-keyout "$PRIVATE_KEY_PATH" \
	-out "$SERVER_CSR_PATH" \
	-subj "/C=MX/ST=Sonora/L=Hermosillo/O=Neuraclinic/OU=Development/CN=localhost" \
	-config "$TMP_CONFIG" \
	-reqexts server_req_ext

openssl x509 \
	-req \
	-in "$SERVER_CSR_PATH" \
	-CA "$CA_CERTIFICATE_PEM_PATH" \
	-CAkey "$CA_PRIVATE_KEY_PATH" \
	-CAserial "$CA_SERIAL_PATH" \
	-CAcreateserial \
	-out "$CERTIFICATE_PATH" \
	-days "$VALID_DAYS" \
	-sha256 \
	-extfile "$TMP_CONFIG" \
	-extensions server_cert_ext

cp "$CA_CERTIFICATE_PEM_PATH" "$CA_CERTIFICATE_CRT_PATH"

chmod 600 "$CA_PRIVATE_KEY_PATH" "$PRIVATE_KEY_PATH"
chmod 644 "$CA_CERTIFICATE_PEM_PATH" "$CA_CERTIFICATE_CRT_PATH" "$CERTIFICATE_PATH"

echo "Generated TLS files in $CERTS_DIR"
