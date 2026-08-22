FROM golang:1.26-alpine AS build

WORKDIR /src

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags conformance -trimpath -ldflags='-s -w' -o /out/graft-docker-runtime-conformance ./cmd/graft-docker-runtime-conformance

FROM alpine:3.22

COPY --from=build /out/graft-docker-runtime-conformance /app/graft-docker-runtime-conformance

ENTRYPOINT ["/app/graft-docker-runtime-conformance"]
