FROM docker:27-cli AS docker-cli

FROM golang:1.26-alpine AS build
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/graft-compose-runner ./cmd/graft-compose-runner

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 65532 graft-runner \
    && adduser -S -D -H -u 65532 -G graft-runner graft-runner
COPY --from=docker-cli /usr/local/bin/docker /usr/local/bin/docker
COPY --from=docker-cli /usr/local/libexec/docker/cli-plugins/docker-compose /usr/local/libexec/docker/cli-plugins/docker-compose
COPY --from=build /out/graft-compose-runner /usr/local/bin/graft-compose-runner
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/graft-compose-runner"]
