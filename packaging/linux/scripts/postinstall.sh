#!/bin/sh

sudo setcap 'cap_dac_read_search=ep' /usr/bin/observe-agent

# Check if systemd is available and handle service restart for upgrades
if [[ -d /run/systemd/system ]]; then
    # Reload systemd daemon to pick up any service file changes
    systemctl daemon-reload

    # For RPM packages: $1 = 1 (install), $1 = 2 (upgrade)
    # For DEB packages: $1 = "configure" (both install and upgrade)

    if systemctl list-unit-files observe-agent.service >/dev/null 2>&1; then
        # Service file exists
        
        # Determine if this is an upgrade or fresh install
        # For RPM: 1 = install, 2+ = upgrade
        # For DEB: we check if service was previously running or enabled
        IS_UPGRADE=false
        if [ "$1" = "2" ] || [ "$1" -gt "2" ] 2>/dev/null; then
            # RPM upgrade
            IS_UPGRADE=true
        elif [ "$1" = "configure" ] && (systemctl is-active --quiet observe-agent.service || systemctl is-enabled --quiet observe-agent.service); then
            # DEB upgrade (service was previously active or enabled)
            IS_UPGRADE=true
        fi
        
        if [ "$IS_UPGRADE" = true ]; then
            # This is an upgrade - ensure service is enabled and restart it
            echo "Ensuring observe-agent.service is enabled after upgrade..."
            systemctl enable observe-agent.service
            echo "Restarting observe-agent.service..."
            systemctl restart observe-agent.service
        else
            # Fresh install - enable and start the service
            echo "Enabling and starting observe-agent.service..."
            systemctl enable --now observe-agent.service
        fi
    fi
fi