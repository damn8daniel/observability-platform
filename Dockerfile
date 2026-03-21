# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /observability-platform ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /observability-platform .
COPY config.yaml .

USER appuser

EXPOSE 8080 9090

ENTRYPOINT ["./observability-platform"]
CMD ["-config", "config.yaml"]
