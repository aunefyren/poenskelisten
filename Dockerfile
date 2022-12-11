FROM golang:1.19-alpine

ENV port=8080

RUN apk update
RUN apk add git

ENV GO111MODULE=on

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build

EXPOSE 8080

ENTRYPOINT /app/poenskelisten 