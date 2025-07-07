FROM golang:1.23.5-bullseye AS builder

# Install Tesseract OCR and development libraries
# Install Tesseract OCR and development libraries with retries
RUN for i in $(seq 1 5); do \
    apt-get update && apt-get install -y \
    tesseract-ocr \
    libtesseract-dev \
    libleptonica-dev \
    pkg-config \
    --no-install-recommends \
    && break || sleep 5; \
done \
&& rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy

# Build with CGO enabled for Tesseract support
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /analyzer ./cmd/api/

# Use Debian slim for runtime to support Tesseract
FROM debian:bullseye-slim

# Install runtime dependencies for Tesseract
RUN apt-get update && apt-get install -y \
    tesseract-ocr \
    --no-install-recommends \
    && rm -rf /var/lib/apt/lists/* \
    && addgroup --system nonroot \
    && adduser --system --ingroup nonroot nonroot

COPY --from=builder /analyzer /analyzer
USER nonroot:nonroot
EXPOSE 8080

ENV GIN_MODE=release

ENTRYPOINT ["/analyzer"]
