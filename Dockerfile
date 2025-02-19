FROM golang:1.21.5-bullseye as builder

ARG TARGETARCH
ARG TARGETOS

WORKDIR /app

COPY . .

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build

FROM debian:bullseye-slim as runtime

LABEL org.opencontainers.image.source=https://github.com/aunefyren/poenskelisten

WORKDIR /app

COPY --from=builder /app .

RUN apt update
RUN apt install -y curl

ENTRYPOINT /app/entrypoint.sh