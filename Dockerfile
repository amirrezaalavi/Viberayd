# Stage 1: Build the VibeRay binary
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache dependencies in a separate layer (go.sum only present when there are external deps)
COPY go.mod go.sum* ./
RUN go mod download

# Copy the rest of the source code and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/viberay ./cmd/viberay

# Stage 2: Minimal runtime image
FROM alpine:3.20 AS runtime

# Install runtime dependencies
RUN apk add --no-cache ca-certificates wget unzip

# Download the official Xray binary (pinned to v26.7.11)
RUN wget -qO /tmp/xray.zip https://github.com/XTLS/Xray-core/releases/download/v26.7.11/Xray-linux-64.zip && \
    unzip -j /tmp/xray.zip xray -d /usr/local/bin/ && \
    chmod +x /usr/local/bin/xray && \
    rm /tmp/xray.zip

# Copy the built VibeRay binary from the builder stage
COPY --from=builder /out/viberay /usr/local/bin/viberay

# Run as non-root user
USER nobody:nobody

WORKDIR /work

ENTRYPOINT ["/usr/local/bin/viberay"]
CMD ["-help"]
