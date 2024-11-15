#!/bin/bash

set -e

service_name="com.observeinc.agent"
observeagent_install_dir="/usr/local/observe-agent"
# TODO download the file instead of using the local built zip
script_dir=$(dirname -- "$(readlink -f -- "$0")")
zip_dir="$(dirname $script_dir)/dist/darwin_arm64_v8.0/observe-agent_Darwin_arm64.zip"

# If the observe-agent.yaml file already exists, leave it alone.
# Otherwise we need to know what the collection endpoint and token are.
if [ ! -f "$observeagent_install_dir/observe-agent.yaml" ]; then
    if [ -z "$OBSERVE_URL" ]; then
        echo "OBSERVE_URL is not set"
        exit 1
    fi

    if [ -z "$TOKEN" ]; then
        echo "TOKEN is not set"
        exit 1
    fi
fi

# TODO download the zip file from the latest release

unzip_dir="/tmp/observe-agent"
rm -rf $unzip_dir
unzip $zip_dir -d $unzip_dir >/dev/null

if [ -f "/Library/LaunchDaemons/$service_name.plist" ]; then
    echo "Stopping $service_name. This may ask for your password..."
    sudo launchctl stop "$service_name" || true
    sudo launchctl unload -wF /Library/LaunchDaemons/$service_name.plist || true
fi

# Create the needed directories
echo "Creating system install folders. This may ask for your password..."
sudo mkdir -p $observeagent_install_dir /usr/local/libexec /usr/local/bin /var/lib/observe-agent/filestorage
sudo chmod +rw /var/lib/observe-agent/filestorage

# Copy all files to the install dir.
sudo cp -f $unzip_dir/observe-agent $observeagent_install_dir/observe-agent
sudo cp -fR $unzip_dir/config $observeagent_install_dir/config
sudo cp -fR $unzip_dir/connections $observeagent_install_dir/connections
sudo chown -R root:wheel $observeagent_install_dir

# Initialize the agent config file if it doesn't exist
if [ -f "$observeagent_install_dir/observe-agent.yaml" ]; then
    echo "Leaving existing observe-agent.yaml in place."
else
    echo "Initializing observe-agent.yaml"
    sudo $observeagent_install_dir/observe-agent init-config --token $TOKEN --observe_url $OBSERVE_URL --config_path $observeagent_install_dir/observe-agent.yaml
    sudo chown root:wheel $observeagent_install_dir/observe-agent.yaml
fi

# Create a link to the agent in the libexec to be used by launchd and bin to be in the user path
sudo ln -sf $observeagent_install_dir/observe-agent /usr/local/libexec/observe-agent
sudo ln -sf $observeagent_install_dir/observe-agent /usr/local/bin

echo "Observe agent successfully installed to $observeagent_install_dir"

# Install the launchd agent
echo "Installing $service_name as a LaunchDaemon. This may ask for your password..."
sudo cp -f $unzip_dir/observe-agent.plist /Library/LaunchDaemons/$service_name.plist
sudo chown root:wheel /Library/LaunchDaemons/$service_name.plist
sudo launchctl load -w /Library/LaunchDaemons/$service_name.plist
sudo launchctl kickstart "system/$service_name"

echo
echo "---"
echo "Installation complete! You can view the agent status with observe-agent status"
echo "Agent logs will be written to /var/log/observe-agent.log"
echo "Use launchctl to stop and start the agent."
exit 0
