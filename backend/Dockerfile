# Step 1: Build Next.js frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /workspace
COPY ../frontend/package*.json ./
RUN npm ci

COPY ../frontend .
RUN npm run build

# Step 2: Build the Go binary
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -o axiom-bridge-server .

# Step 3: Final minimal image
FROM node:20-alpine

# Install dumb-init for proper signal handling
RUN apk add --no-cache dumb-init ca-certificates

WORKDIR /app

# Copy Go binary
COPY --from=builder /app/axiom-bridge-server /app/
COPY public /app/public

# Copy Next.js frontend
COPY --from=frontend-builder /workspace/.next /app/.next
COPY --from=frontend-builder /workspace/node_modules /app/node_modules
COPY --from=frontend-builder /workspace/package.json /app/

ENV PORT=8080
EXPOSE 8080

# Start with dumb-init to handle signals properly
ENTRYPOINT ["dumb-init", "--"]

# Run Go backend on :8080
CMD ["./axiom-bridge-server"]

