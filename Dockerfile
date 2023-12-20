FROM golang:1.20.4-bullseye as builder

ARG TARGETARCH
ARG TARGETOS

WORKDIR /app

COPY . .

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build

FROM debian:bullseye-slim as runtime

LABEL org.opencontainers.image.source=https://github.com/aunefyren/poenskelisten

ENV port=8080
ENV timezone=Europe/Oslo
ENV dbip=localhost
ENV dbport=3306
ENV dbname=poenskelisten
ENV dbusername=root
ENV dbpassword=root
ENV generateinvite=false
ENV disablesmtp=false
ENV smtphost=smtp.gmail.com
ENV smtpport=25
ENV smtpusername=mycoolusernameformysmtpserver@justanexample.org
ENV smtppassword=password123

WORKDIR /app

COPY --from=builder /app .

ENTRYPOINT /app/poenskelisten -port ${port} -timezone ${timezone} -generateinvite ${generateinvite}-dbip ${dbip}-dbport ${dbport} -dbname ${dbname} -dbusername ${dbusername} -dbpassword ${dbpassword} -disablesmtp ${disablesmtp} -smtphost ${smtphost} -smtpport ${smtpport} -smtpusername ${smtpusername} -smtppassword ${smtppassword}