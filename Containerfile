FROM --platform=$BUILDPLATFORM docker.io/library/alpine:3.24.2 AS certificates
RUN apk add --no-cache ca-certificates

FROM docker.io/library/alpine:3.24.2

ARG TARGETPLATFORM
ENV GIN_MODE=release
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /app
COPY ${TARGETPLATFORM}/no_trainer /usr/local/bin/no_trainer
USER 65532:65532
EXPOSE 8080
CMD ["/usr/local/bin/no_trainer"]
