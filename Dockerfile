FROM golang:1.19-alpine

LABEL org.opencontainers.image.source=https://github.com/aunefyren/poenskelisten

ENV port=8080
ENV timezone=Europe/Oslo
ENV dbip=localhost
ENV dbport=3306
ENV dbname=poenskelisten
ENV dbusername=root
ENV dbpassword=root

RUN apk update
RUN apk add git

ENV GO111MODULE=on

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build

ENTRYPOINT /app/poenskelisten -port ${port} -timezone ${timezone} -dbip ${dbip} -dbport ${dbport} -dbname ${dbname} -dbusername ${dbusername} -dbpassword ${dbpassword}