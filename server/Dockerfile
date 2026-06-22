FROM golang:1.26-bookworm AS builder

WORKDIR /src/server

COPY go.mod go.sum ./
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}
RUN go mod download

COPY . .

ARG GRAFT_VERSION=dev
ARG GRAFT_GIT_COMMIT=unknown
ARG GRAFT_BUILD_TIME_UTC=unknown
ARG GRAFT_GIT_TREE_STATE=dirty

ENV CGO_ENABLED=0

RUN go build \
    -trimpath \
    -ldflags="-s -w \
      -X 'graft/server/internal/buildinfo.version=${GRAFT_VERSION}' \
      -X 'graft/server/internal/buildinfo.gitCommit=${GRAFT_GIT_COMMIT}' \
      -X 'graft/server/internal/buildinfo.buildTimeUTC=${GRAFT_BUILD_TIME_UTC}' \
      -X 'graft/server/internal/buildinfo.gitTreeState=${GRAFT_GIT_TREE_STATE}'" \
    -o /out/graft \
    ./cmd/graft

FROM alpine:3.22 AS runtime

RUN apk add --no-cache ca-certificates curl \
    && addgroup -S -g 10001 graft \
    && adduser -S -D -H -u 10001 -G graft graft

WORKDIR /app

COPY --from=builder /out/graft /app/graft

ENV GRAFT_HTTP_ADDR=:8080

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=15s --retries=5 CMD curl --fail --silent http://127.0.0.1:8080/healthz >/dev/null || exit 1

USER graft:graft

ENTRYPOINT ["/app/graft"]
CMD ["serve"]
