FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/graft-docker-builder-agent ./agents/docker-builder-agent/cmd

FROM docker:27-cli
RUN addgroup -S graft && adduser -S -G graft graft && mkdir -p /etc/graft/config /var/lib/graft-builder-agent/state && chown -R graft:graft /etc/graft /var/lib/graft-builder-agent
COPY --from=build /out/graft-docker-builder-agent /app/graft-docker-builder-agent
USER graft
VOLUME ["/etc/graft/config", "/var/lib/graft-builder-agent/state"]
ENTRYPOINT ["/app/graft-docker-builder-agent"]
