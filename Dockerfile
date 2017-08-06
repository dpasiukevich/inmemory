FROM golang:latest 

RUN mkdir /inmemory
ADD . /inmemory
ADD . /go/src/github.com/dpasiukevich/inmemory
WORKDIR /inmemory/server
RUN go build server.go 