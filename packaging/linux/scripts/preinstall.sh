#!/bin/sh

getent passwd observe-agent >/dev/null || useradd --system --user-group --no-create-home --shell /sbin/nologin observe-agent
