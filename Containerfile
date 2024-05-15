# vim: set ft=Dockerfile

#############################
# STEP 1 prepare other files
############################
FROM alpine AS builder

RUN apk --update --no-cache add ca-certificates

# Create appuser.
ENV USER=notraineruser
ENV UID=10001
# See https://stackoverflow.com/a/55757473/12429735RUN
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/home/${USER}" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

############################
# STEP 2 build a small image
############################
FROM scratch

# Import from builder.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy our static executable.
COPY no_trainer /go/bin/no_trainer

# Use an unprivileged user.
USER notraineruser:notraineruser

VOLUME ["/home/notraineruser"]

# Run the binary.
ENTRYPOINT ["/go/bin/no_trainer"]