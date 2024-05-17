FROM golang:alpine

ENV GIN_MODE=release
COPY no_trainer /go/bin/no_trainer
CMD ["/go/bin/no_trainer"]
