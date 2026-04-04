# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web/frontend
COPY web/frontend/package*.json ./
RUN npm ci
COPY web/frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.24-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/frontend/dist ./web/frontend/dist
RUN CGO_ENABLED=1 go build -o nianhe ./cmd/nianhe

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend /app/nianhe .
EXPOSE 8080
VOLUME /app/data
ENV NIANHE_DB=/app/data/nianhe.db
CMD ["./nianhe"]
