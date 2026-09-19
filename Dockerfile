# ---------- Build ----------
# Runs on the build host's own platform and cross-compiles (CGO is off), so the
# multi-arch images don't compile under QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
WORKDIR /app

# faster caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# TARGETVARIANT is "v7" for linux/arm/v7 (GOARM=7) and empty elsewhere.
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} \
    go build -trimpath -ldflags="-s -w" -o /app/poenskelisten .

# ---------- Runtime ----------
FROM alpine:3.24
# PUID/PGID are read at container start by entrypoint.sh, which drops from root
# to that uid/gid, so they can be overridden with runtime environment variables.
ENV PUID=1000 PGID=1000 LANG=C.UTF-8 LC_ALL=C.UTF-8
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata su-exec
COPY --from=builder /app/poenskelisten /app/poenskelisten
COPY --from=builder /app/entrypoint.sh /app/entrypoint.sh
COPY --from=builder /app/web/ /app/web/
RUN chmod +x /app/poenskelisten /app/entrypoint.sh && \
    mkdir -p /app/files /app/images
EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]
