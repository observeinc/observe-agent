FROM alpine:3.20.0 as directories

RUN mkdir -p /var/lib/observe-agent/filestorage

FROM alpine:3.20.0 as certs
RUN apk --update add ca-certificates

FROM debian:12.5 as systemd
RUN apt update && apt install -y systemd
# prepare package with journald and it's dependencies keeping original paths
# h stands for dereference of symbolic links
RUN tar czhf journalctl.tar.gz /bin/journalctl $(ldd /bin/journalctl | grep -oP "\/.*? ")
# extract package to /output so it can be taken as base for scratch image
# we do not copy archive into scratch image, as it doesn't have tar executable
# however, we can copy full directory as root (/) to be base file structure for scratch image
RUN mkdir /output && tar xf /journalctl.tar.gz --directory /output


FROM scratch

# ARG USER_UID=10001
# USER ${USER_UID}

ADD packaging/docker/observe-agent /etc/observe-agent
COPY --from=systemd /output/ /
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
# COPY --from=directories --chown=${USER_UID}:${USER_UID} /var/lib/observe-agent/filestorage /var/lib/observe-agent/filestorage
COPY --from=directories /var/lib/observe-agent/filestorage /var/lib/observe-agent/filestorage
COPY agent /

ENTRYPOINT ["/agent"]
CMD ["start"]
# RUN apk --update add ca-certificates

# RUN mkdir -p /tmp

# FROM scratch

# ARG USER_UID=10001
# USER ${USER_UID}

# COPY --from=prep /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
# COPY otelcontribcol /
# EXPOSE 4317 55680 55679
# ENTRYPOINT ["/otelcontribcol"]
# CMD ["--config", "/etc/otel/config.yaml"]