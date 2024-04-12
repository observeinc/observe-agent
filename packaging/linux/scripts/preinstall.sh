#!/bin/sh

# Set up user and permissions
getent passwd observe-agent >/dev/null || useradd --system --user-group --no-create-home --shell /sbin/nologin observe-agent
sudo adduser observe-agent systemd-journal

sudo mkdir -p /var/lib/otelcol/file_storage/receiver
sudo chown observe-agent /var/lib/otelcol/file_storage/receiver 
