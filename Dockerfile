FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /analyzer ./cmd/api/

FROM alpine:latest
RUN addgroup -S nonroot && adduser -S nonroot -G nonroot
COPY --from=builder /analyzer /analyzer
USER nonroot:nonroot
EXPOSE 8080

ENV GIN_MODE=release

ENTRYPOINT ["/analyzer"]