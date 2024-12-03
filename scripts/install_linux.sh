#!/bin/bash

set -e

agent_binary_path="/usr/bin/observe-agent"
observeagent_config_dir="/etc/observe-agent"
tmp_dir="/tmp/observe-agent"

# If the observe-agent.yaml file already exists, leave it alone.
# Otherwise we need to know what the collection endpoint and token are.
if [ ! -f "$observeagent_config_dir/observe-agent.yaml" ]; then
    if [ -z "$OBSERVE_URL" ]; then
        echo "OBSERVE_URL env var is not set"
        exit 1
    fi

    if [ -z "$TOKEN" ]; then
        echo "TOKEN env var is not set"
        exit 1
    fi
fi

# If the zip file is not provided, download the latest release from GitHub.
if [ -z "$ZIP_DIR" ]; then
    echo "Downloading latest release from GitHub"
    curl -L -o /tmp/observe-agent.tar.gz https://github.com/observeinc/observe-agent/releases/latest/download/observe-agent_Linux_$(arch).tar.gz
    ZIP_DIR="/tmp/observe-agent.tar.gz"
else
    echo "Installing from provided zip file: $ZIP_DIR"
fi

# Set up the temp dir and extract files.
rm -rf $tmp_dir
mkdir -p $tmp_dir
tar -xzf $ZIP_DIR -C $tmp_dir

# Create the needed directories.
echo "Creating system install folders. This may ask for your password..."
sudo mkdir -p $observeagent_config_dir /var/lib/observe-agent/filestorage
sudo chmod +rw /var/lib/observe-agent/filestorage

# Move the binary to the proper path.
cp -f $tmp_dir/observe-agent $agent_binary_path

# Copy all config files to the proper dir.
sudo cp -f $tmp_dir/otel-collector.yaml $observeagent_config_dir/otel-collector.yaml
sudo rm -rf $observeagent_config_dir/connections
sudo cp -fR $tmp_dir/connections $observeagent_config_dir/connections
sudo chown -R root:root $observeagent_config_dir

# Initialize the agent config file if it doesn't exist.
if [ -f "$observeagent_config_dir/observe-agent.yaml" ]; then
    echo "Leaving existing observe-agent.yaml in place."
else
    echo "Initializing observe-agent.yaml"
    sudo $agent_binary_path init-config --token $TOKEN --observe_url $OBSERVE_URL --config_path $observeagent_config_dir/observe-agent.yaml
    sudo chown root:root $observeagent_config_dir/observe-agent.yaml
fi

echo "Observe agent successfully installed to $agent_binary_path with config in $observeagent_config_dir"
