#!/bin/sh

# For RPM packages: $1 = 0 (uninstall), $1 = 1 (upgrade) 
# For DEB packages: $1 = "purge" (uninstall), $1 = "upgrade" (upgrade)

if command -v systemctl >/dev/null 2>&1; then
    # Check if this is an upgrade (not a complete uninstall)
    if [ "$1" = "1" ] || [ "$1" = "upgrade" ]; then
        # Upgrade - restart the service that was stopped by the old version's preremove
        if systemctl is-enabled --quiet observe-agent.service; then
            echo "Restarting observe-agent.service after upgrade cleanup..."
            systemctl start observe-agent.service
        fi
    fi
    # For complete uninstall ($1 = "0" or "purge"), do nothing - service should stay stopped
fi