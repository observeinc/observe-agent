#!/bin/sh

echo "[PREREMOVE] Running with arg: '$1'"
echo "[PREREMOVE] Script version: NEW (with debug)"

# Check current service state before any action
if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet observe-agent.service; then
        echo "[PREREMOVE] Service is currently ACTIVE"
    else
        echo "[PREREMOVE] Service is currently INACTIVE"
    fi
    
    if systemctl is-enabled --quiet observe-agent.service; then
        echo "[PREREMOVE] Service is currently ENABLED"
    else
        echo "[PREREMOVE] Service is currently DISABLED"
    fi
fi

# For RPM packages: $1 = 0 (uninstall), $1 = 1 (upgrade) 
# For DEB packages: $1 = "remove" (uninstall), $1 = "upgrade" (upgrade)
# Only stop and disable the service on complete uninstall, not during upgrades

if command -v systemctl >/dev/null 2>&1; then
    # Check if this is an uninstall (not an upgrade)
    if [ "$1" = "0" ] || [ "$1" = "remove" ]; then
        # Complete uninstall - stop and disable the service
        echo "[PREREMOVE] Detected uninstall (arg=$1)"
        echo "[PREREMOVE] Stopping and disabling observe-agent.service..."
        systemctl stop observe-agent.service
        systemctl disable observe-agent.service
        echo "[PREREMOVE] Service stopped and disabled"
    elif [ "$1" = "1" ] || [ "$1" = "upgrade" ]; then
        # Upgrade - only stop the service, don't disable it
        echo "[PREREMOVE] Detected upgrade (arg=$1)"
        echo "[PREREMOVE] Stopping observe-agent.service for upgrade (NOT disabling)..."
        systemctl stop observe-agent.service
        echo "[PREREMOVE] Service stopped but NOT disabled"
        
        # Check if service is still enabled after preremove
        if systemctl is-enabled --quiet observe-agent.service; then
            echo "[PREREMOVE] Service is still ENABLED after preremove"
        else
            echo "[PREREMOVE] WARNING: Service is DISABLED after preremove"
        fi
    else
        echo "[PREREMOVE] Unknown argument: '$1' - taking no action"
    fi
else
    echo "[PREREMOVE] systemctl not available"
fi

echo "[PREREMOVE] Script completed"