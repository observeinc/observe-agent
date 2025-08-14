#!/bin/bash

set -e

service_name="com.observeinc.agent"
observeagent_install_dir="/usr/local/observe-agent"
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
        --setup_launch_daemon)
            SETUP_LAUNCH_DAEMON="$arg"
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

# If the zip file is not provided, download the latest release from GitHub.
if [ -z "$ZIP_DIR" ]; then
    if [ -n "$AGENT_VERSION" ]; then
        if [[ $AGENT_VERSION =~ ^[[:digit:]] ]]; then
            AGENT_VERSION="v$AGENT_VERSION"
        fi
        echo "Downloading version $AGENT_VERSION from GitHub..."
        curl -s -L -o /tmp/observe-agent.zip https://github.com/observeinc/observe-agent/releases/download/$AGENT_VERSION/observe-agent_Darwin_$(arch).zip
    else
        echo "Downloading latest release from GitHub..."
        curl -s -L -o /tmp/observe-agent.zip https://github.com/observeinc/observe-agent/releases/latest/download/observe-agent_Darwin_$(arch).zip
    fi
    ZIP_DIR="/tmp/observe-agent.zip"
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
unzip $ZIP_DIR -d $tmp_dir >/dev/null

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
sudo rm -rf $observeagent_install_dir/config $observeagent_install_dir/connections $observeagent_install_dir/observe-agent
sudo cp $tmp_dir/observe-agent $observeagent_install_dir/observe-agent
if [ -d $tmp_dir/config ]; then
    sudo cp -R $tmp_dir/config $observeagent_install_dir/config
fi
if [ -d $tmp_dir/connections ]; then
    sudo cp -R $tmp_dir/connections $observeagent_install_dir/connections
fi
sudo chown -R root:wheel $observeagent_install_dir

# Initialize the agent config file if it doesn't exist
if [ -f "$observeagent_install_dir/observe-agent.yaml" ]; then
    echo "Leaving existing observe-agent.yaml in place."
else
    echo "Initializing observe-agent.yaml"
    INIT_FLAGS="--config_path $observeagent_install_dir/observe-agent.yaml --token $TOKEN --observe_url $OBSERVE_URL --host_monitoring::enabled=true"
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
    # Add default args
    INIT_FLAGS="$INIT_FLAGS --forwarding::metrics::output_format=true"
    sudo $observeagent_install_dir/observe-agent init-config $INIT_FLAGS
    sudo chown root:wheel $observeagent_install_dir/observe-agent.yaml
fi

# Create a link to the agent in the libexec to be used by launchd and bin to be in the user path
sudo ln -sf $observeagent_install_dir/observe-agent /usr/local/libexec/observe-agent
sudo ln -sf $observeagent_install_dir/observe-agent /usr/local/bin

echo "Observe agent successfully installed to $observeagent_install_dir"

# Install the launchd agent unless the variable is specified to false
if [ -z "$SETUP_LAUNCH_DAEMON" ] || [[ "$(echo "$SETUP_LAUNCH_DAEMON" | tr '[:upper:]' '[:lower:]')" == "true" ]]; then
    echo "Installing $service_name as a LaunchDaemon. This may ask for your password..."
    sudo cp -f $tmp_dir/observe-agent.plist /Library/LaunchDaemons/$service_name.plist
    sudo chown root:wheel /Library/LaunchDaemons/$service_name.plist
    sudo launchctl load -w /Library/LaunchDaemons/$service_name.plist
    sudo launchctl kickstart "system/$service_name"
fi

echo
echo "---"
echo "Installation complete! You can view the agent status with observe-agent status"
echo "Agent logs will be written to /var/log/observe-agent.log"
echo "Use launchctl to stop and start the agent."
exit 0
