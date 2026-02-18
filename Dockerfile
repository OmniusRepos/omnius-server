FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/../static/admin /app/static/admin
ARG VERSION=dev
ARG COMMIT=""
RUN if [ "$VERSION" = "dev" ] && [ -d .git ]; then \
      VERSION=$(git describe --tags --always 2>/dev/null || echo "dev"); \
    fi && \
    if [ -z "$COMMIT" ] && [ -d .git ]; then \
      COMMIT=$(git rev-parse HEAD 2>/dev/null || echo ""); \
    fi && \
    CGO_ENABLED=0 go build -ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT}" -o main .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main /app/main
RUN mkdir -p /app/data
COPY --from=builder /app/static /app/static
COPY --from=builder /app/templates /app/templates
EXPOSE 8080
CMD ["/app/main"]
