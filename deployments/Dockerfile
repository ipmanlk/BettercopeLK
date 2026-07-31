# Multi-stage build for BettercopeLK
# Stage 1: build the binary
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /build/bettercopelk ./cmd/bettercopelk

# Stage 2: minimal runtime image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /build/bettercopelk .
COPY --from=builder /build/sources ./sources
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s \
  CMD wget -qO- http://localhost:8080/healthz || exit 1
ENTRYPOINT ["/app/bettercopelk"]
