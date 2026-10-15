#!/bin/sh
# Upgrades ZITADEL (API and login UI together) to VERSION on this server.
#
#   scripts/upgrade.sh v4.19.5
#
# Takes a backup first: ZITADEL migrations cannot be undone, so rolling back
# means restoring that backup with the old version (docs/runbooks/upgrade.md).
set -eu
cd "$(dirname "$0")/.."
. scripts/lib.sh

new=${1:?usage: upgrade.sh VERSION (e.g. v4.19.5)}
case "$new" in
v[0-9]*.[0-9]*.[0-9]*) ;;
*)
    echo "version must look like v4.19.5" >&2
    exit 2
    ;;
esac
: "${IDENTITY_DOMAIN:?upgrade.sh is for staging and production; in dev edit ZITADEL_VERSION and run make dev-up}"
old=${ZITADEL_VERSION:?}
if [ "$old" = "$new" ]; then
    echo "already on $new"
    exit 0
fi
echo "upgrading ZITADEL $old -> $new ($IDENTITY_ENV)"

docker pull "ghcr.io/zitadel/zitadel:$new"
ui_version=$(cat frontend/upstream-version.txt)
if [ "$ui_version" != "$new" ]; then
    echo "login source is $ui_version; update it to $new per frontend/UPSTREAM.md and rerun the auth checks first" >&2
    exit 1
fi
ZITADEL_VERSION=$new compose build login

scripts/backup.sh

start=$(date +%s)
sed -i.bak "s/^ZITADEL_VERSION=.*/ZITADEL_VERSION=$new/" "$env_file"
rm -f "$env_file.bak"
ZITADEL_VERSION=$new compose up -d --wait

issuer="https://$IDENTITY_DOMAIN"
got=$(curl -fsS "$issuer/.well-known/openid-configuration" | sed -n 's/.*"issuer":"\([^"]*\)".*/\1/p')
if [ "$got" != "$issuer" ]; then
    echo "discovery answered issuer '$got', expected $issuer; investigate before going on" >&2
    exit 1
fi
curl -fsS "$issuer/debug/ready" >/dev/null
echo "upgraded to $new in $(($(date +%s) - start))s; now run the checks in docs/runbooks/upgrade.md"
