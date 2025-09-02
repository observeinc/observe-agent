#!/bin/sh

echo "[POSTREMOVE] Running with arg: '$1'"
echo "[POSTREMOVE] Script version: NEW (with service fix)"

# Check current service state
if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet observe-agent.service; then
        echo "[POSTREMOVE] Service is currently ACTIVE"
    else
        echo "[POSTREMOVE] Service is currently INACTIVE"
    fi
    
    if systemctl is-enabled --quiet observe-agent.service; then
        echo "[POSTREMOVE] Service is currently ENABLED"
    else
        echo "[POSTREMOVE] Service is currently DISABLED"
    fi
fi

# For RPM packages: $1 = 0 (uninstall), $1 = 1 (upgrade) 
# For DEB packages: $1 = "purge" (uninstall), $1 = "upgrade" (upgrade)

if command -v systemctl >/dev/null 2>&1; then
    # Check if this is an upgrade (not a complete uninstall)
    if [ "$1" = "1" ] || [ "$1" = "upgrade" ]; then
        echo "[POSTREMOVE] This is an upgrade - checking if service needs to be re-enabled..."
        
        # Check if service is enabled
        if systemctl is-enabled --quiet observe-agent.service; then
            echo "[POSTREMOVE] Service is already enabled"
            echo "[POSTREMOVE] Restarting observe-agent.service after upgrade cleanup..."
            systemctl start observe-agent.service
            
            # Check if start was successful
            sleep 1
            if systemctl is-active --quiet observe-agent.service; then
                echo "[POSTREMOVE] Service successfully restarted and is now ACTIVE"
            else
                echo "[POSTREMOVE] WARNING: Service failed to start after postremove"
                # Show status for debugging
                systemctl status observe-agent.service --no-pager -l
            fi
        else
            echo "[POSTREMOVE] WARNING: Service is not enabled, cannot restart"
        fi
    elif [ "$1" = "0" ] || [ "$1" = "purge" ]; then
        echo "[POSTREMOVE] This is a complete uninstall - leaving service stopped"
    else
        echo "[POSTREMOVE] Unknown argument: '$1'"
    fi
else
    echo "[POSTREMOVE] systemctl not available"
fi

echo "[POSTREMOVE] Script completed"