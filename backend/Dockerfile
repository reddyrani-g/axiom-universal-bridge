# Build the Go binary
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -o axiom-bridge-server .

# Final minimal image
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
# Copy binary
COPY --from=builder /app/axiom-bridge-server .

ENV PORT=8080
EXPOSE 8080

CMD ["./axiom-bridge-server"]

