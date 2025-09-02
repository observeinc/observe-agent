#!/bin/sh

# For RPM packages: $1 = 0 (uninstall), $1 = 1 (upgrade) 
# For DEB packages: $1 = "remove" (uninstall), $1 = "upgrade" (upgrade)
# Only stop and disable the service on complete uninstall, not during upgrades

if command -v systemctl >/dev/null 2>&1; then
    # Check if this is an uninstall (not an upgrade)
    if [ "$1" = "0" ] || [ "$1" = "remove" ]; then
        # Complete uninstall - stop and disable the service
        echo "Stopping and disabling observe-agent.service..."
        systemctl stop observe-agent.service
        systemctl disable observe-agent.service
    elif [ "$1" = "1" ] || [ "$1" = "upgrade" ]; then
        # Upgrade - only stop the service, don't disable it
        echo "Stopping observe-agent.service for upgrade..."
        systemctl stop observe-agent.service
    fi
fi