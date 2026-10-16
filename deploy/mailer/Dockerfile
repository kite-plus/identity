# Build context: the repository root (compose sets it).
FROM golang:1.26.4-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/identity-mailer ./cmd/identity-mailer

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/identity-mailer /identity-mailer
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 CMD ["/identity-mailer", "-healthcheck"]
ENTRYPOINT ["/identity-mailer"]
