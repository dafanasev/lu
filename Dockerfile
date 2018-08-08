FROM golang:alpine AS builder
COPY . /go/src/github.com/dafanasev/lu
WORKDIR /go/src/github.com/dafanasev/lu
RUN apk add --no-cache git \
    && go get ./... \
    && go get -u github.com/gobuffalo/packr/... \
    && packr \
    && go build \
    && apk del git

FROM alpine
MAINTAINER Dmitrii Afanasev <dimarzio1986@gmail.com>
RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/dafanasev/lu/lu /
CMD ["/lu"]