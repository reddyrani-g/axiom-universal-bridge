# Step 1: Build Next.js frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /workspace/frontend
COPY frontend/package*.json ./
RUN npm ci

COPY frontend .
RUN npm run build

# Step 2: Build the Go binary
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend .
# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -o axiom-bridge-server .

# Step 3: Final image
FROM node:20-alpine

# Install system dependencies
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy Go binary
COPY --from=builder /app/axiom-bridge-server .

# Copy Next.js frontend build
COPY --from=frontend-builder /workspace/frontend/.next /app/.next
COPY --from=frontend-builder /workspace/frontend/node_modules /app/node_modules
COPY --from=frontend-builder /workspace/frontend/package.json /app/package.json

# Copy public files if they exist
COPY backend/public /app/public 2>/dev/null || true

ENV PORT=8080
EXPOSE 8080

# Run the Go server
CMD ["./axiom-bridge-server"]
