FROM golang:1.22.11-alpine3.21 AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /analyzer ./cmd/api/

FROM alpine:latest
COPY --from=builder /analyzer /analyzer
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/analyzer"]