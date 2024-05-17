FROM golang:alpine

COPY no_trainer /go/bin/no_trainer
CMD ["/go/bin/no_trainer"]
