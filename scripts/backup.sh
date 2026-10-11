#!/bin/sh
# Dumps the ZITADEL database and the bootstrap tokens into one archive
# encrypted with age, then prunes archives older than BACKUP_KEEP_DAYS.
# The master key is deliberately not included; it is kept offline.
#
#   scripts/backup.sh                    # production layout (deploy/.env)
#   IDENTITY_ENV=dev scripts/backup.sh   # the local identity-dev stack
#
# Needs: docker, age. Settings come from the env file: BACKUP_AGE_RECIPIENT
# (age public key), BACKUP_DIR, optional BACKUP_KEEP_DAYS (default 30).
set -eu
cd "$(dirname "$0")/.."
. scripts/lib.sh

: "${BACKUP_AGE_RECIPIENT:?set BACKUP_AGE_RECIPIENT (an age public key)}"
: "${BACKUP_DIR:?set BACKUP_DIR}"
keep=${BACKUP_KEEP_DAYS:-30}

ts=$(date -u +%Y%m%dT%H%M%SZ)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

compose exec -T db pg_dump -U postgres -d zitadel -Fc --no-owner >"$work/zitadel.dump"
cp -R "$BOOTSTRAP_DIR" "$work/bootstrap"
printf 'zitadel_version=%s\ncreated=%s\nenvironment=%s\n' \
    "$(zitadel_version)" "$ts" "$IDENTITY_ENV" >"$work/MANIFEST"

mkdir -p "$BACKUP_DIR"
out="$BACKUP_DIR/identity-$IDENTITY_ENV-$ts.tar.age"
tar -C "$work" -cf - MANIFEST zitadel.dump bootstrap | age -r "$BACKUP_AGE_RECIPIENT" -o "$out.tmp"
mv "$out.tmp" "$out"
chmod 600 "$out"
echo "backup written: $out ($(wc -c <"$out" | tr -d ' ') bytes)"

find "$BACKUP_DIR" -name "identity-$IDENTITY_ENV-*.tar.age" -mtime +"$keep" -print -delete
