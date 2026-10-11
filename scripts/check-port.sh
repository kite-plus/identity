#!/bin/sh
# Fails early with a clear message when the host port is taken by something
# other than the given compose project.
set -eu

port=$1
project=$2

owner=$(docker ps --filter "publish=$port" --format '{{.Label "com.docker.compose.project"}} {{.Names}}' | head -n1)
case "$owner" in
"$project "*) exit 0 ;;
"") ;;
*)
    echo "port $port is published by container ${owner#* } (compose project ${owner%% *})." >&2
    echo "Stop it, or set IDENTITY_DEV_PORT in dev/.env and run make dev-reset." >&2
    exit 1
    ;;
esac

if lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "port $port is in use on this machine; set IDENTITY_DEV_PORT in dev/.env." >&2
    exit 1
fi
