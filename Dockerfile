FROM golang:1.19-bullseye

LABEL org.opencontainers.image.source=https://github.com/aunefyren/poenskelisten

ARG TARGETARCH 
ARG TARGETOS 

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

COPY . .

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build 

ENTRYPOINT /app/poenskelisten -port ${port} -timezone ${timezone} -generateinvite ${generateinvite} -dbip ${dbip} -dbport ${dbport} -dbname ${dbname} -dbusername ${dbusername} -dbpassword ${dbpassword} -disablesmtp ${disablesmtp} -smtphost ${smtphost} -smtpport ${smtpport} -smtpusername ${smtpusername} -smtppassword ${smtppassword}