#!/bin/sh

echo "[POSTINSTALL] Running with arg: '$1'"
echo "[POSTINSTALL] Script version: NEW (with extensive debug)"

sudo setcap 'cap_dac_read_search=ep' /usr/bin/observe-agent

# Check if systemd is available and handle service restart for upgrade
if [[ -d /run/systemd/system ]]; then
    echo "[POSTINSTALL] Systemd detected"
    # Reload systemd daemon to pick up any service file changes
    systemctl daemon-reload
    echo "[POSTINSTALL] Daemon reloaded"

    # For RPM packages: $1 = 1 (install), $1 = 2 (upgrade)
    # For DEB packages: $1 = "configure" (both install and upgrade)

    if systemctl list-unit-files observe-agent.service >/dev/null 2>&1; then
        echo "[POSTINSTALL] Service file exists"
        
        # Check current service state
        if systemctl is-active --quiet observe-agent.service; then
            echo "[POSTINSTALL] Service is currently ACTIVE"
        else
            echo "[POSTINSTALL] Service is currently INACTIVE"
        fi
        
        if systemctl is-enabled --quiet observe-agent.service; then
            echo "[POSTINSTALL] Service is currently ENABLED"
        else
            echo "[POSTINSTALL] Service is currently DISABLED"
        fi
        
        # Determine if this is an upgrade or fresh install
        # For RPM: 1 = install, 2+ = upgrade
        # For DEB: we check if service was previously running or enabled
        IS_UPGRADE=false
        if [ "$1" = "2" ] || [ "$1" -gt "2" ] 2>/dev/null; then
            # RPM upgrade
            IS_UPGRADE=true
            echo "[POSTINSTALL] Detected upgrade (RPM arg=$1)"
        elif [ "$1" = "configure" ] && (systemctl is-active --quiet observe-agent.service || systemctl is-enabled --quiet observe-agent.service); then
            # DEB upgrade (service was previously active or enabled)
            IS_UPGRADE=true
            echo "[POSTINSTALL] Detected upgrade (DEB configure with active/enabled service)"
        else
            echo "[POSTINSTALL] Detected fresh install (arg=$1)"
        fi
        
        if [ "$IS_UPGRADE" = true ]; then
            # This is an upgrade - ensure service is enabled and restart it
            echo "[POSTINSTALL] Processing upgrade - ensuring service is enabled..."
            systemctl enable observe-agent.service
            echo "[POSTINSTALL] Service enabled, now restarting..."
            systemctl restart observe-agent.service
            echo "[POSTINSTALL] Service restart command completed"
            
            # Check if restart was successful
            sleep 1
            if systemctl is-active --quiet observe-agent.service; then
                echo "[POSTINSTALL] Service is now ACTIVE after restart"
            else
                echo "[POSTINSTALL] WARNING: Service is still INACTIVE after restart"
            fi
        else
            # Fresh install - enable and start the service
            echo "[POSTINSTALL] Processing fresh install - enabling and starting service..."
            systemctl enable --now observe-agent.service
            echo "[POSTINSTALL] Enable --now command completed"
            
            # Check if start was successful
            sleep 1
            if systemctl is-active --quiet observe-agent.service; then
                echo "[POSTINSTALL] Service is now ACTIVE after fresh start"
            else
                echo "[POSTINSTALL] WARNING: Service is still INACTIVE after fresh start"
            fi
        fi
    else
        echo "[POSTINSTALL] WARNING: Service file not found"
    fi
else
    echo "[POSTINSTALL] Systemd not detected"
fi

echo "[POSTINSTALL] Script completed"