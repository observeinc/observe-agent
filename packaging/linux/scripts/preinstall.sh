#!/bin/sh

getent passwd observe-agent >/dev/null || useradd --system --user-group --no-create-home --shell /sbin/nologin observe-agent

sudo mkdir /var/lib/otelcol
sudo mkdir /var/lib/otelcol/file_storage
sudo chown -R observe-agent /var/lib/otelcol/file_storage