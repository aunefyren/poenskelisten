# syntax=docker/dockerfile:1
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

# GOARM comes from the variant (linux/arm/v7 -> 7) and only applies to GOARCH=arm;
# arm64 can carry a "v8" variant that must not leak into GOARM.
RUN if [ "${TARGETARCH}" = "arm" ]; then export GOARM="${TARGETVARIANT#v}"; fi && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /app/poenskelisten .

# ---------- Runtime ----------
FROM alpine:3.24
# PUID/PGID are read at container start by entrypoint.sh, which drops from root
# to that uid/gid, so they can be overridden with runtime environment variables.
ENV PUID=1000 PGID=1000 LANG=C.UTF-8 LC_ALL=C.UTF-8
WORKDIR /app
# The only RUN in this stage, since each one runs under QEMU on the non-native
# platforms; the executable bits are set by COPY --chmod instead.
RUN apk add --no-cache ca-certificates tzdata su-exec && \
    mkdir -p /app/files /app/images
COPY --from=builder --chmod=755 /app/poenskelisten /app/poenskelisten
COPY --from=builder --chmod=755 /app/entrypoint.sh /app/entrypoint.sh
COPY --from=builder /app/web/ /app/web/
EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]
