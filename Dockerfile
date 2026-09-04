# Stage 1: Build viberayd
FROM golang:1.21-alpine AS builder

WORKDIR /src

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/viberayd ./cmd/viberayd

# Stage 2: Runtime
FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates wget unzip

RUN wget -qO /tmp/xray.zip https://github.com/XTLS/Xray-core/releases/download/v26.7.11/Xray-linux-64.zip && \
    unzip -j /tmp/xray.zip xray -d /usr/local/bin/ && \
    chmod +x /usr/local/bin/xray && \
    rm /tmp/xray.zip

COPY --from=builder /out/viberayd /usr/local/bin/viberayd

USER nobody:nobody

WORKDIR /work

EXPOSE 8080 8081

ENTRYPOINT ["/usr/local/bin/viberayd"]
