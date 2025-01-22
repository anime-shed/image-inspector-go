# Build stage
FROM golang:1.23.5-bookworm AS builder

WORKDIR /app

# Install OpenCV dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    pkg-config \
    libopencv-dev \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /analyzer ./cmd/api/

# Runtime stage
FROM gcr.io/distroless/base-debian12

# Install runtime dependencies
COPY --from=builder /usr/lib/x86_64-linux-gnu/libopencv* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libgomp.so.1 /usr/lib/x86_64-linux-gnu/
COPY --from=builder /analyzer /analyzer

# Create non-root user
RUN addgroup --gid 65532 nonroot && \
    adduser --disabled-password --gecos "" --uid 65532 --ingroup nonroot nonroot

USER nonroot:nonroot

EXPOSE 8080
ENTRYPOINT ["/analyzer"]