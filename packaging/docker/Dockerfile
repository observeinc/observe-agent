# For FIPS binary, there are some debian runtime dependencies
FROM debian:12.5 AS agent
COPY observe-agent /
# This shouldn't be necessary but sometimes we end up with execution bit not set.
# ref: https://github.com/open-telemetry/opentelemetry-collector/issues/1317
RUN chmod 755 /observe-agent
# The base executable includes symlinks to libraries. In order to break those, 
# we tar with the h flag to dereference symlinks. 
RUN tar czhf observe-agent.tar.gz /observe-agent $(ldd /observe-agent | grep -oP "\/.*? ")
# extract package to /output so it can be taken as base for scratch image
# we do not copy archive into scratch image, as it doesn't have tar executable
# however, we can copy full directory as root (/) to be base file structure for scratch image
RUN mkdir /output && tar xf /observe-agent.tar.gz --directory /output

FROM alpine:3.20.2 AS directories
RUN mkdir -p /var/lib/observe-agent/filestorage

FROM alpine:3.20.2 AS certs
RUN apk --update add ca-certificates

FROM debian:12.5 AS systemd
RUN apt update && apt install -y systemd
# prepare package with journald and it's dependencies keeping original paths
# h stands for dereference of symbolic links
RUN tar czhf journalctl.tar.gz /bin/journalctl $(ldd /bin/journalctl | grep -oP "\/.*? ")
# extract package to /output so it can be taken as base for scratch image
# we do not copy archive into scratch image, as it doesn't have tar executable
# however, we can copy full directory as root (/) to be base file structure for scratch image
RUN mkdir /output && tar xf /journalctl.tar.gz --directory /output

FROM alpine:3.20.2
ADD packaging/docker/observe-agent /etc/observe-agent
COPY --from=systemd /output/ /
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=agent /output/ /
COPY --from=directories /var/lib/observe-agent/filestorage /var/lib/observe-agent/filestorage

EXPOSE 4317 55680 55679 13133
ENTRYPOINT ["/observe-agent"]
CMD ["start"]
