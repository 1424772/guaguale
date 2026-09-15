FROM golang:1.26-alpine AS builder

WORKDIR /workspace
COPY apps/api/go.mod ./
RUN go mod download

COPY apps/api ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/server /app/server

USER app
EXPOSE 8080
ENTRYPOINT ["/app/server"]

