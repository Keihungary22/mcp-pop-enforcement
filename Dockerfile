# Build stage / Build用Stage
FROM golang:1.26.5-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/pop-enforcement \
    ./cmd/enforcement


# Runtime stage / 実行用Stage
FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/pop-enforcement /app/pop-enforcement

USER app

EXPOSE 9100

ENTRYPOINT ["/app/pop-enforcement"]
