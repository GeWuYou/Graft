FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags conformance -trimpath -ldflags='-s -w' -o /out/graft-docker-runtime-conformance ./cmd/graft-docker-runtime-conformance

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && addgroup -S graft && adduser -S -G graft graft && mkdir -p /conformance/agent-bootstrap /conformance/agent-config && chown -R graft:graft /conformance
COPY --from=build /out/graft-docker-runtime-conformance /usr/local/bin/graft-docker-runtime-conformance
USER graft
ENTRYPOINT ["/usr/local/bin/graft-docker-runtime-conformance"]
