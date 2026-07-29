FROM golang:1.26-alpine AS build
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/graft-compose-runner ./cmd/graft-compose-runner

FROM docker:27-cli
COPY --from=build /out/graft-compose-runner /usr/local/bin/graft-compose-runner
USER 0:0
ENTRYPOINT ["/usr/local/bin/graft-compose-runner"]
