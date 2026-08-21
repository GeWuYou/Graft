FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/graft-docker-runtime-agent ./agents/docker-runtime-agent/cmd

FROM docker:27-cli
RUN addgroup -S graft && adduser -S -G graft graft && mkdir -p /etc/graft/config /var/lib/graft-runtime-agent/state && chown -R graft:graft /etc/graft /var/lib/graft-runtime-agent
COPY --from=build /out/graft-docker-runtime-agent /app/graft-docker-runtime-agent
USER graft
VOLUME ["/etc/graft/config", "/var/lib/graft-runtime-agent/state"]
HEALTHCHECK --interval=15s --timeout=5s --retries=5 CMD ["/app/graft-docker-runtime-agent", "--healthcheck"]
ENTRYPOINT ["/app/graft-docker-runtime-agent"]
