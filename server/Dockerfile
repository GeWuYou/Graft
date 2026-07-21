FROM alpine:3.22 AS runtime

RUN apk add --no-cache ca-certificates curl \
    && addgroup -S -g 10001 graft \
    && adduser -S -D -H -u 10001 -G graft graft

WORKDIR /app

COPY --from=release-binary /graft-server-linux-amd64 /app/graft

ENV GRAFT_HTTP_ADDR=:8080

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=15s --retries=5 CMD curl --fail --silent http://127.0.0.1:8080/healthz >/dev/null || exit 1

USER graft:graft

ENTRYPOINT ["/app/graft"]
CMD ["serve"]
