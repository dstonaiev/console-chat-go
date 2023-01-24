FROM golang:alpine as build-env

ENV GO111MODULE=on

RUN apk update && apk add bash ca-certificates git gcc g++ libc-dev

RUN mkdir /chat
#RUN mkdir -p /chat/proto 

WORKDIR /chat

COPY . .

COPY go.mod .
COPY go.sum .

RUN go mod download

RUN go build ./cmd/server

ENTRYPOINT ["./chat-server", "-p", "17100"]