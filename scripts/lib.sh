# Shared helpers for the operations scripts; sourced, not executed.
# IDENTITY_ENV=production|staging (default production, both use deploy/)
# or dev (the local identity-dev stack). IDENTITY_PROJECT overrides the
# compose project name, e.g. for a restore drill next to a running stack.
# Compose runs inside the project directory, as an operator would run it,
# so COMPOSE_FILE and COMPOSE_PROJECT_NAME from the env file (edge mode,
# staging next to production) apply here too.

IDENTITY_ENV=${IDENTITY_ENV:-production}

case "$IDENTITY_ENV" in
dev)
    env_file=${IDENTITY_ENV_FILE:-dev/.env}
    project_dir=dev
    ;;
production | staging)
    env_file=${IDENTITY_ENV_FILE:-deploy/.env}
    project_dir=deploy
    ;;
*)
    echo "unknown IDENTITY_ENV $IDENTITY_ENV" >&2
    exit 2
    ;;
esac

[ -f "$env_file" ] || {
    echo "missing $env_file" >&2
    exit 2
}
env_file=$(cd "$(dirname "$env_file")" && pwd)/$(basename "$env_file")
set -a
. "$env_file"
set +a

if [ "$IDENTITY_ENV" = dev ]; then
    BOOTSTRAP_DIR=${BOOTSTRAP_DIR:-dev/out/bootstrap}
else
    case "${SECRETS_DIR:=./secrets}" in
    /*) BOOTSTRAP_DIR=${BOOTSTRAP_DIR:-$SECRETS_DIR/bootstrap} ;;
    *) BOOTSTRAP_DIR=${BOOTSTRAP_DIR:-$project_dir/$SECRETS_DIR/bootstrap} ;;
    esac
fi

compose() {
    (
        cd "$project_dir" || exit
        if [ -n "${IDENTITY_PROJECT:-}" ]; then
            docker compose -p "$IDENTITY_PROJECT" --env-file "$env_file" "$@"
        else
            docker compose --env-file "$env_file" "$@"
        fi
    )
}

zitadel_version() {
    compose images zitadel --format json 2>/dev/null | sed -n 's/.*"Tag":"\([^"]*\)".*/\1/p' | head -n1
}
