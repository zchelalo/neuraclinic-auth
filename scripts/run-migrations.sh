#!/bin/sh
set -eu

: "${DB_HOST:?missing DB_HOST}"
: "${DB_USER:?missing DB_USER}"
: "${DB_PASS:?missing DB_PASS}"
: "${DB_NAME:?missing DB_NAME}"
: "${DB_PORT:?missing DB_PORT}"

MIGRATIONS_PATH="${MIGRATIONS_PATH:-/app/migrations}"
DB_SSLMODE="${DB_SSLMODE:-disable}"

export PGPASSWORD="$DB_PASS"

DATABASE_URL="postgres://${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"
output_file="$(mktemp)"

if migrate -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" up >"$output_file" 2>&1; then
	cat "$output_file"
	rm -f "$output_file"
	exit 0
fi

if grep -qi "no change" "$output_file"; then
	cat "$output_file"
	rm -f "$output_file"
	exit 0
fi

cat "$output_file" >&2
rm -f "$output_file"
exit 1
