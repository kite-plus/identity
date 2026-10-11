# Shared helpers for the operations scripts; sourced, not executed.
# IDENTITY_ENV=production|staging (default production, both use deploy/)
# or dev (the local identity-dev stack). IDENTITY_PROJECT overrides the
# compose project name, e.g. for a restore drill next to a running stack.

IDENTITY_ENV=${IDENTITY_ENV:-production}

case "$IDENTITY_ENV" in
dev)
    env_file=${IDENTITY_ENV_FILE:-dev/.env}
    compose_file=dev/docker-compose.yaml
    project_dir=dev
    ;;
production | staging)
    env_file=${IDENTITY_ENV_FILE:-deploy/.env}
    compose_file=deploy/docker-compose.yaml
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
set -a
. "./$env_file"
set +a

if [ "$IDENTITY_ENV" = dev ]; then
    BOOTSTRAP_DIR=${BOOTSTRAP_DIR:-dev/out/bootstrap}
else
    BOOTSTRAP_DIR=${BOOTSTRAP_DIR:-$project_dir/${SECRETS_DIR:-./secrets}/bootstrap}
fi

compose() {
    if [ -n "${IDENTITY_PROJECT:-}" ]; then
        docker compose -p "$IDENTITY_PROJECT" --project-directory "$project_dir" -f "$compose_file" --env-file "$env_file" "$@"
    else
        docker compose --project-directory "$project_dir" -f "$compose_file" --env-file "$env_file" "$@"
    fi
}

zitadel_version() {
    compose images zitadel --format json 2>/dev/null | sed -n 's/.*"Tag":"\([^"]*\)".*/\1/p' | head -n1
}
