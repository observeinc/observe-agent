#!/bin/bash

set -e

agent_binary_path="/usr/bin/observe-agent"
observeagent_config_dir="/etc/observe-agent"
tmp_dir="/tmp/observe-agent"

# Parse args
while [ $# -gt 0 ]; do
    opt=$1
    shift
    arg=""
    if [[ "$opt" == *"="* ]]; then
        arg=$(echo $opt | cut -d'=' -f2)
        opt=$(echo $opt | cut -d'=' -f1)
    elif [ $# -gt 0 ]; then
        arg="$1"
        shift
    fi
    case "$opt" in
        --token)
            TOKEN="$arg"
            ;;
        --observe_url)
            OBSERVE_URL="$arg"
            ;;
        --logs_enabled)
            LOGS_ENABLED="$arg"
            ;;
        --metrics_enabled)
            METRICS_ENABLED="$arg"
            ;;
        --version)
            AGENT_VERSION="$arg"
            ;;
        --zip_dir)
            ZIP_DIR="$arg"
            ;;
        *)
            echo "Unknown option: $opt"
            exit 1
            ;;
    esac
done

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
    if [ -n "$AGENT_VERSION" ]; then
        echo "Downloading version $AGENT_VERSION from GitHub..."
        curl -s -L -o /tmp/observe-agent.tar.gz https://github.com/observeinc/observe-agent/releases/download/v$AGENT_VERSION/observe-agent_Linux_$(arch).tar.gz
    else
        echo "Downloading latest release from GitHub..."
        curl -s -L -o /tmp/observe-agent.tar.gz https://github.com/observeinc/observe-agent/releases/latest/download/observe-agent_Linux_$(arch).tar.gz
    fi
    ZIP_DIR="/tmp/observe-agent.tar.gz"
else
    if [ -n "$AGENT_VERSION" ]; then
        echo "Cannot specify both ZIP_DIR ($ZIP_DIR) and AGENT_VERSION ($AGENT_VERSION)"
        exit 1
    fi
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
sudo cp -f $tmp_dir/observe-agent $agent_binary_path

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
    INIT_FLAGS="--config_path $observeagent_config_dir/observe-agent.yaml --token $TOKEN --observe_url $OBSERVE_URL --host_monitoring::enabled=true"
    if [ -n "$LOGS_ENABLED" ]; then
        if [[ "$(echo "$LOGS_ENABLED" | tr '[:upper:]' '[:lower:]')" == "true" ]]; then
            INIT_FLAGS="$INIT_FLAGS --host_monitoring::logs::enabled=true"
        else
            INIT_FLAGS="$INIT_FLAGS --host_monitoring::logs::enabled=false"
        fi
    fi
    if [ -n "$METRICS_ENABLED" ]; then
        if [[ "$(echo "$METRICS_ENABLED" | tr '[:upper:]' '[:lower:]')" == "true" ]]; then
            INIT_FLAGS="$INIT_FLAGS --host_monitoring::metrics::host::enabled=true"
        else
            INIT_FLAGS="$INIT_FLAGS --host_monitoring::metrics::host::enabled=false"
        fi
    fi
    sudo $agent_binary_path init-config $INIT_FLAGS
    sudo chown root:root $observeagent_config_dir/observe-agent.yaml
fi

# Check if systemd is available.
if [[ -d /run/systemd/system ]]; then
    # Set up the systemd service if it doesn't exist already.
    if ! systemctl list-unit-files observe-agent.service | grep observe-agent >/dev/null; then
        echo "Installing observe-agent.service as a systemd service. This may ask for your password..."

        # Set up user and permissions (copied from preinstall.sh)
        sudo getent passwd observe-agent >/dev/null || sudo useradd --system --user-group --no-create-home --shell /sbin/nologin observe-agent
        sudo usermod -a -G systemd-journal observe-agent
        sudo mkdir -p /var/lib/observe-agent/filestorage
        sudo chown -R observe-agent:observe-agent /var/lib/observe-agent/filestorage

        # Copy the service file and start the service.
        sudo cp -f $tmp_dir/observe-agent.service /etc/systemd/system/observe-agent.service
        sudo chown root:root /etc/systemd/system/observe-agent.service
        sudo systemctl daemon-reload
        sudo systemctl enable observe-agent.service
        sudo systemctl start observe-agent
    elif systemctl is-active --quiet observe-agent; then
        echo "Restarting observe-agent.service. This may ask for your password..."
        sudo systemctl restart observe-agent
    fi
fi

echo "Observe agent successfully installed to $agent_binary_path with config in $observeagent_config_dir"
