# ---------- Build ----------
FROM golang:1.24.0-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app

# faster caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /app/poenskelisten .

# ---------- Runtime ----------
FROM alpine:3.20
ENV PUID=1000 PGID=1000 LANG=C.UTF-8 LC_ALL=C.UTF-8
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/poenskelisten /app/poenskelisten
COPY --from=builder /app/entrypoint.sh /app/entrypoint.sh
COPY --from=builder /app/web/ /app/web/
RUN addgroup -g ${PGID} appgroup && \
    adduser -D -u ${PUID} -G appgroup appuser && \
    chmod +x /app/poenskelisten /app/entrypoint.sh && \
    chown -R appuser:appgroup /app
USER appuser
ENTRYPOINT ["/app/entrypoint.sh"]