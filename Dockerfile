# Не меняй этот файл: по нему панель собирает и узнаёт образы сервиса.
# Сборка идёт на платформе раннера и кросс-компилирует под целевую: образы нужны и под amd64 (сервер), и под arm64 (Mac).
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
# CMD — папка бинаря в cmd/: api, workers/<имя> или cron/<имя>. Образ называется так же.
ARG CMD
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    test -n "$CMD" && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=$VERSION" -o /out/app ./cmd/$CMD

FROM --platform=$BUILDPLATFORM alpine:3.21 AS goose
RUN apk add --no-cache ca-certificates curl
ARG TARGETARCH
ARG GOOSE_VERSION=v3.25.0
RUN set -eu; \
    case "$TARGETARCH" in \
      amd64) asset="goose_linux_x86_64" ;; \
      arm64) asset="goose_linux_arm64" ;; \
      *) echo "unsupported TARGETARCH=$TARGETARCH"; exit 1 ;; \
    esac; \
    base="https://github.com/pressly/goose/releases/download/${GOOSE_VERSION}"; \
    curl -fsSL -o /goose "${base}/${asset}"; \
    curl -fsSL -o /checksums.txt "${base}/checksums.txt"; \
    want="$(awk -v a="$asset" '$2 == a {print $1}' /checksums.txt)"; \
    test -n "$want"; \
    echo "${want}  /goose" | sha256sum -c -; \
    chmod +x /goose

FROM gcr.io/distroless/static:nonroot AS service
COPY --from=build /out/app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]

FROM gcr.io/distroless/static:nonroot AS migrator
COPY --from=goose /goose /goose
COPY migrations /migrations
USER nonroot:nonroot
ENTRYPOINT ["/goose"]
CMD ["up"]
