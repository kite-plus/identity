#!/bin/sh
# Restores a backup made by scripts/backup.sh into an empty stack.
#
#   scripts/restore.sh BACKUP.tar.age AGE_IDENTITY_FILE
#
# The env file must hold the same ZITADEL_MASTERKEY as the backed-up
# instance (the dump is useless without it) and the same domain. Database
# passwords may differ: the zitadel role is created here with the
# configured ZITADEL_DB_PASSWORD. Refuses to touch an existing database.
# Same IDENTITY_ENV / IDENTITY_PROJECT settings as backup.sh.
set -eu
cd "$(dirname "$0")/.."
. scripts/lib.sh

archive=${1:?usage: restore.sh BACKUP.tar.age AGE_IDENTITY_FILE}
identity=${2:?usage: restore.sh BACKUP.tar.age AGE_IDENTITY_FILE}

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
age -d -i "$identity" "$archive" | tar -C "$work" -xf -
cat "$work/MANIFEST"

backup_version=$(sed -n 's/^zitadel_version=//p' "$work/MANIFEST")
want_version=${ZITADEL_VERSION:-$backup_version}
if [ -n "$backup_version" ] && [ "$backup_version" != "$want_version" ]; then
    echo "backup is from ZITADEL $backup_version but ZITADEL_VERSION is $want_version." >&2
    echo "Restore with the backup's version first, then upgrade (docs/runbooks/upgrade.md)." >&2
    exit 1
fi

compose up -d --wait db

exists=$(compose exec -T db psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname = 'zitadel'")
if [ "$exists" = 1 ]; then
    echo "database zitadel already exists in this stack; restore only into an empty one." >&2
    exit 1
fi

compose exec -T db psql -U postgres -v ON_ERROR_STOP=1 -v pw="$ZITADEL_DB_PASSWORD" <<'SQL'
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'zitadel') THEN
    CREATE ROLE zitadel LOGIN;
  END IF;
END $$;
ALTER ROLE zitadel PASSWORD :'pw';
CREATE DATABASE zitadel OWNER zitadel;
SQL

# Objects are recreated as the zitadel role so it owns them, as after a
# fresh `zitadel init`.
compose exec -T db pg_restore -U postgres -d zitadel --no-owner --role=zitadel --exit-on-error <"$work/zitadel.dump"

mkdir -p "$BOOTSTRAP_DIR"
cp "$work"/bootstrap/* "$BOOTSTRAP_DIR"/
chmod 600 "$BOOTSTRAP_DIR"/*

compose up -d --wait
echo "restored $(sed -n 's/^created=//p' "$work/MANIFEST") into $IDENTITY_ENV${IDENTITY_PROJECT:+ ($IDENTITY_PROJECT)}"
