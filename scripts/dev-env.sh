#!/bin/sh
# Creates or completes dev/.env with random secrets for the identity-dev
# stack. Existing values are kept, so the file survives `make dev-reset`.
# IDENTITY_DEV_PORT picks the host port on first run (default 8091).
set -eu

env_file=${1:-dev/.env}
umask 077
[ -f "$env_file" ] || printf '# Secrets for the local identity-dev stack. Do not commit.\n' >"$env_file"

rand() { openssl rand -hex "$1"; }

add() {
    grep -q "^$1=" "$env_file" || printf '%s=%s\n' "$1" "$2" >>"$env_file"
}

add IDENTITY_DEV_PORT "${IDENTITY_DEV_PORT:-8091}"
add ZITADEL_MASTERKEY "$(rand 16)"
add POSTGRES_ADMIN_PASSWORD "$(rand 16)"
add ZITADEL_DB_PASSWORD "$(rand 16)"
add ADMIN_PASSWORD "Admin-$(rand 8)"
add LOGIN_SESSION_COOKIE_SECRET "$(rand 32)"
add ALICE_PASSWORD "alice-$(rand 6)"
add BOB_PASSWORD "bob-$(rand 6)"
