#!/bin/bash

set -e

service_name="com.observeinc.agent"
observeagent_install_dir="/usr/local/observe-agent"

echo "Uninstalling Observe Agent..."

# Stop and remove the LaunchDaemon
echo "Stopping $service_name. This may ask for your password..."
sudo launchctl bootout "system/$service_name" 2>/dev/null || true
sudo rm -f /Library/LaunchDaemons/$service_name.plist

# Remove symlinks
sudo rm -f /usr/local/libexec/observe-agent
sudo rm -f /usr/local/bin/observe-agent

# Remove the binary but leave the install dir and config for potential reinstall
sudo rm -f $observeagent_install_dir/observe-agent
sudo rm -rf $observeagent_install_dir/config $observeagent_install_dir/connections

# Remove data directories and logs
sudo rm -rf /var/lib/observe-agent
sudo rm -f /var/log/observe-agent.log

echo
echo "---"
echo "Observe Agent has been uninstalled."
exit 0
