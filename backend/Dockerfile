# Step 1: Build the Go binary
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -o axiom-bridge-server .

# Step 2: Final minimal image
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
# Copy binary
COPY --from=builder /app/axiom-bridge-server .
# Copy frontend public folder from local build
COPY public/ ./public/

ENV PORT=8080
EXPOSE 8080

CMD ["./axiom-bridge-server"]
