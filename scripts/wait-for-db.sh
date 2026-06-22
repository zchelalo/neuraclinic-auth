#!/usr/bin/env sh
set -eu

DB_HOST="${1:?missing DB_HOST}"
DB_PORT="${2:-5432}"

until nc -z -v -w30 "$DB_HOST" "$DB_PORT"; do
	echo "Waiting for database at $DB_HOST:$DB_PORT..."
	sleep 1
done

echo "Database is up and running at $DB_HOST:$DB_PORT"
