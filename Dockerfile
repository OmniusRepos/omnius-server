FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN if [ "$VERSION" = "dev" ] && [ -d .git ]; then \
      VERSION=$(git describe --tags --always 2>/dev/null || echo "dev"); \
    fi && \
    CGO_ENABLED=0 go build -ldflags "-X main.Version=${VERSION}" -o main .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main /app/main
RUN mkdir -p /app/data
COPY --from=builder /app/static /app/static
COPY --from=builder /app/templates /app/templates
EXPOSE 8080
CMD ["/app/main"]
