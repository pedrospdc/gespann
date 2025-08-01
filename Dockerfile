# Build stage
FROM golang:1.24.5-alpine AS builder

# Install eBPF build dependencies
RUN apk add --no-cache \
    clang \
    llvm \
    libbpf-dev \
    linux-headers \
    make \
    gcc \
    musl-dev \
    git \
    bpftool

WORKDIR /src

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Initialize git repo to avoid version error
RUN git init && git config user.email "docker@build" && git config user.name "Docker Build"

# Generate vmlinux.h from available kernel headers
RUN if [ -f /sys/kernel/btf/vmlinux ]; then \
        bpftool btf dump file /sys/kernel/btf/vmlinux format c > bpf/vmlinux.h; \
    else \
        echo "Using fallback vmlinux.h (BTF not available in container)"; \
    fi

# Generate eBPF code and build
RUN make generate && make build

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    libbpf

WORKDIR /app

# Copy binary from builder
COPY --from=builder /src/bin/gespann .
COPY --from=builder /src/config.yaml .

# Expose Prometheus metrics port
EXPOSE 8080

# Run as root (required for eBPF)
USER root

ENTRYPOINT ["./gespann"]
CMD ["-config", "config.yaml"]