#!/bin/sh

# Set up user and permissions
getent passwd observe-agent >/dev/null || useradd --system --user-group --no-create-home --shell /sbin/nologin observe-agent
sudo usermod -a -G systemd-journal observe-agent

sudo mkdir -p /var/lib/observe-agent/filestorage
sudo mkdir -p /var/lib/observe-agent/data
sudo chown -R observe-agent:observe-agent /var/lib/observe-agent
sudo chmod 1777 /var/lib/observe-agent/filestorage
sudo chmod 1777 /var/lib/observe-agent/data
