FROM node:24-alpine AS build
WORKDIR /src
RUN npm install --global pnpm@10.30.3
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/login/package.json apps/login/
COPY packages/zitadel-client/package.json packages/zitadel-client/
COPY packages/zitadel-proto/package.json packages/zitadel-proto/
RUN --mount=type=cache,target=/pnpm/store pnpm install --frozen-lockfile --store-dir /pnpm/store
COPY . .
ENV NEXT_PUBLIC_BASE_PATH=/ui/v2/login NEXT_TELEMETRY_DISABLED=1
RUN pnpm build

FROM node:24-alpine AS runtime
WORKDIR /app
RUN addgroup --system --gid 1001 nodejs && adduser --system --uid 1001 nextjs
COPY --from=build --chown=nextjs:nodejs /src/apps/login/.next/standalone ./
ENV HOSTNAME=0.0.0.0 PORT=3000 NODE_ENV=production \
    NODE_OPTIONS="--use-openssl-ca --require /app/load-ssl-cert-dir.cjs" \
    SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt ZITADEL_TLS_ENABLED=false OTEL_SDK_DISABLED=true
USER nextjs
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD ["node", "/app/healthcheck.mjs", "http://localhost:3000/ui/v2/login/ready"]
ENTRYPOINT ["/app/entrypoint.sh", "node", "apps/login/server.js"]
