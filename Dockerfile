# Stage 1: Build
FROM golang:1.25.2-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o seeder cmd/seed/main.go

# Stage 2: Final Image
FROM alpine:latest

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/main .
COPY --from=builder /app/seeder .
COPY --from=builder /app/migrations ./migrations

# Install goose for migrations
RUN apk add --no-cache ca-certificates
ADD https://github.com/pressly/goose/releases/download/v3.20.0/goose_linux_x86_64 /usr/local/bin/goose
RUN chmod +x /usr/local/bin/goose

EXPOSE 8080

CMD ["./main"]
