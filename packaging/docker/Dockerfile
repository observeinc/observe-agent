# We need systemd binaries to be able to run the journald receiver
FROM debian:12.13 AS systemd
RUN apt update && apt install -y systemd
# prepare package with journald and it's dependencies keeping original paths
# h stands for dereference of symbolic links
RUN tar czhf journalctl.tar.gz /bin/journalctl $(ldd /bin/journalctl | grep -oP "\/.*? ")
# extract package to /output so it can be taken as base for scratch image
# we do not copy archive into scratch image, as it doesn't have tar executable
# however, we can copy full directory as root (/) to be base file structure for scratch image
RUN mkdir /output && tar xf /journalctl.tar.gz --directory /output

FROM alpine:3.23.3 AS certs
RUN apk --update add ca-certificates

FROM alpine:3.23.3

ARG USER_UID=10001
ARG USER_GID=10001

RUN mkdir -p /var/lib/observe-agent /etc/observe-agent/
RUN chown ${USER_UID}:${USER_GID} /var/lib/observe-agent /etc/observe-agent/

USER ${USER_UID}:${USER_GID}

ARG TARGETPLATFORM
COPY --chmod=755 $TARGETPLATFORM/observe-agent /

ADD --chmod=644 --chown=${USER_UID}:${USER_GID} packaging/docker/observe-agent /etc/observe-agent
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=systemd /output/ /

RUN mkdir /var/lib/observe-agent/filestorage /var/lib/observe-agent/data


EXPOSE 4317 4318 55680 55679 13133
ENTRYPOINT ["/observe-agent"]
CMD ["start"]
