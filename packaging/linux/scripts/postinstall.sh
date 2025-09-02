#!/bin/sh

sudo setcap 'cap_dac_read_search=ep' /usr/bin/observe-agent

# Check if systemd is available and handle service restart for upgrades
if command -v systemctl >/dev/null 2>&1; then
    # Reload systemd daemon to pick up any service file changes
    systemctl daemon-reload

    # For RPM packages: $1 = 1 (install), $1 = 2 (upgrade)
    # For DEB packages: $1 = "configure" (both install and upgrade)
    # We'll check if the service exists and is active to determine if this is an upgrade

    if systemctl list-unit-files observe-agent.service >/dev/null 2>&1; then
        # Service file exists
        if systemctl is-active --quiet observe-agent.service; then
            # Service is running, this is likely an upgrade
            echo "Restarting observe-agent.service after upgrade..."
            systemctl restart observe-agent.service
        else
            # Service is not running
            if ! systemctl is-enabled --quiet observe-agent.service; then
                # Service is not enabled, this is likely a fresh install
                echo "Enabling and starting observe-agent.service..."
                systemctl enable --now observe-agent.service
            else
                # Service is enabled but not running, just start it
                echo "Starting observe-agent.service..."
                systemctl start observe-agent.service
            fi
        fi
    fi
fi


