FROM golang:1.24-alpine AS builder
WORKDIR /app
# Cache buster: 20260209a
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o main .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main /app/main
RUN mkdir -p /app/data
COPY --from=builder /app/static /app/static
COPY --from=builder /app/templates /app/templates
EXPOSE 8080
CMD ["/app/main"]
