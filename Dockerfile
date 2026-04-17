FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/token-aggregator ./

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 app \
 && mkdir -p /data && chown app:app /data
WORKDIR /app
COPY --from=builder /out/token-aggregator /app/token-aggregator
USER app
VOLUME ["/data"]
ENV ADDR=":8080" \
    DB_PATH="/data/usage.db" \
    TIMEZONE="Asia/Shanghai"
EXPOSE 8080
ENTRYPOINT ["/app/token-aggregator"]
