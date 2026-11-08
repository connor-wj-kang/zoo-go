FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY admin ./admin
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/zoo-api ./cmd/api

FROM alpine:3.21

RUN adduser -D -u 10001 app \
    && mkdir -p /app/data \
    && chown -R app:app /app
COPY --from=builder /out/zoo-api /usr/local/bin/zoo-api
COPY --from=builder /src/admin /app/admin

USER app
WORKDIR /app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/zoo-api"]
CMD ["-addr", ":8080", "-data", "/app/data/zoo.db", "-uploads", "/app/data/uploads", "-admin", "/app/admin"]
