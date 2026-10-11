DEV_COMPOSE := docker compose --project-directory dev -f dev/docker-compose.yaml --env-file dev/.env
# Loads dev/.env into the environment of the command that follows.
DEV_ENV := set -a && . ./dev/.env && set +a

.PHONY: help dev-up dev-init dev-down dev-reset dev-logs dev-ps dev-test test build lint

help:
	@echo "make dev-up     start the local identity provider (identity-dev)"
	@echo "make dev-init   apply config/dev.yaml and write dev/out/*.env"
	@echo "make dev-down   stop it, keeping data"
	@echo "make dev-reset  delete all data, start again and re-init"
	@echo "make dev-test   end-to-end check against the running stack"
	@echo "make test       unit tests"

dev/.env:
	@scripts/dev-env.sh dev/.env

dev-up: dev/.env
	@scripts/dev-env.sh dev/.env
	@mkdir -p dev/out/bootstrap
	@$(DEV_ENV) && scripts/check-port.sh "$$IDENTITY_DEV_PORT" identity-dev
	$(DEV_COMPOSE) up -d --wait
	@$(DEV_ENV) && echo "identity provider: http://localhost:$$IDENTITY_DEV_PORT  mail: http://localhost:8025"

dev-init: dev/.env
	@$(DEV_ENV) && go run ./cmd/identity-init -config config/dev.yaml -token-file dev/out/bootstrap/admin.pat -out dev/out
	@$(DEV_ENV) && if [ "$$IDENTITY_DEV_PORT" != 8090 ]; then \
		printf 'The dev issuer is http://localhost:%s, not the default http://localhost:8090,\nbecause IDENTITY_DEV_PORT in identity/dev/.env says so (port 8090 was taken).\nRead the issuer from the *.env files here.\n' "$$IDENTITY_DEV_PORT" > dev/out/NOTICE.txt; \
		echo "note: issuer port is $$IDENTITY_DEV_PORT (see dev/out/NOTICE.txt)"; \
	else rm -f dev/out/NOTICE.txt; fi

dev-down:
	$(DEV_COMPOSE) down

dev-reset:
	$(DEV_COMPOSE) down -v
	rm -rf dev/out
	$(MAKE) dev-up dev-init

dev-logs:
	$(DEV_COMPOSE) logs -f --tail 100

dev-ps:
	$(DEV_COMPOSE) ps

dev-test:
	@$(DEV_ENV) && go test -count=1 -tags e2e ./test/e2e/ -v

test:
	go test ./...

build:
	go build -o bin/identity-init ./cmd/identity-init

lint:
	go vet ./...
	go vet -tags e2e ./test/e2e/
