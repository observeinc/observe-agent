#!/bin/sh

if command -v systemctl >/dev/null 2>&1; then
    systemctl stop observe-agent.service
    systemctl disable observe-agent.service
fi
