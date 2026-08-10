FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags conformance -trimpath -ldflags='-s -w' -o /out/graft-docker-builder-conformance ./cmd/graft-docker-builder-conformance

FROM alpine:3.22

COPY --from=build /out/graft-docker-builder-conformance /app/graft-docker-builder-conformance

ENTRYPOINT ["/app/graft-docker-builder-conformance"]
