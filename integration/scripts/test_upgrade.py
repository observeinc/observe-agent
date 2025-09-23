#!/usr/bin/env python3
import os
import sys
import re
import time
import utils as u
from collections import defaultdict


def _check_status_loop(
    remote_host: u.Host, status_command: str, num_retries: int = 10, sleep_seconds: float = 1.5
) -> bool:
    """Run Check Status Command in a loop to wait for observe-agent to start

    Args:
        remote_host (Host): instance to ssh into
        status_command (str): windows/linux status command to run
        num_retries (int): number of times to check for the running agent before giving up
        sleep_seconds (float): number of seconds to sleep between each retry

    Returns:
        bool: agent_status
    """
    agent_status = False
    time.sleep(sleep_seconds)
    for _ in range(num_retries):
        metrics_dict = defaultdict(list)
        try:
            result = remote_host.run_command(status_command)
        except Exception as e:
            print("Ignoring exception: ", e)
            time.sleep(sleep_seconds)
            continue
        for line in result.stdout.splitlines():
            if ":" in line:
                metric, value = line.split(":", 1)
                metric = metric.strip()
                value = value.strip()
                metrics_dict[metric].append(value)
            print(line)
        if metrics_dict["Status"] and metrics_dict["Status"][0] == "Running":
            print("✅ Observe Agent is active and running without errors!")
            agent_status = True
            break
        print(
            "❌ Observe Agent is not running. Retry Count is {}/{}...".format(
                _ + 1, num_retries
            )
        )
        time.sleep(sleep_seconds)
    return agent_status


def _get_agent_version(remote_host: u.Host, version_command: str) -> str:
    """Get the current version of the observe-agent

    Args:
        remote_host (Host): instance to ssh into
        version_command (str): command to get version

    Returns:
        str: version string
    """
    try:
        result = remote_host.run_command(version_command)
        if result.exited == 0:
            # Extract version from output - format is usually "observe-agent version X.Y.Z"
            version_line = result.stdout.strip()
            version_match = re.search(r'version\s+(\S+)', version_line)
            if version_match:
                return version_match.group(1)
        return "unknown"
    except Exception as e:
        print(f"Error getting version: {e}")
        return "unknown"


@u.print_test_decorator
def run_test_windows(remote_host: u.Host, env_vars: dict) -> None:
    """
    Test to upgrade observe-agent on Windows from older version to current build

    Args:
        remote_host (Host): instance to ssh into
        env_vars (dict): environment variables passed into for testing
    """
    # Commands for Windows
    install_old_command = r'powershell -Command "& { [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; Invoke-WebRequest -Uri \"https://github.com/observeinc/observe-agent/releases/download/v2.5.0/observe-agent_Windows_x86_64.zip\" -OutFile \"observe-agent-old.zip\"; Expand-Archive -Path \"observe-agent-old.zip\" -DestinationPath \"observe-agent-old\" -Force; ./observe-agent-old/install.ps1 }"'
    start_command = r".\start_agent_windows.ps1"
    status_command = r'Get-Service ObserveAgent;Set-Location "${Env:Programfiles}\Observe\observe-agent"; ./observe-agent status'
    version_command = r'Set-Location "${Env:Programfiles}\Observe\observe-agent"; ./observe-agent version'
    stop_command = r"Stop-Service ObserveAgent"

    # Get windows home dir paths
    home_dir = r"/C:/Users/{}".format(env_vars["user"])

    print("🔄 Installing older version of observe-agent...")
    result = remote_host.run_command(install_old_command)
    u.print_remote_result(result)
    if result.stderr:
        raise RuntimeError("❌ Error installing older version of observe-agent")

    print("🚀 Starting observe-agent service...")
    # Copy and run start script
    current_script_dir = os.path.dirname(os.path.abspath(__file__))
    ps_start_script_path = os.path.join(current_script_dir, "start_agent_windows.ps1")
    remote_host.put_file(local_path=ps_start_script_path, remote_path=home_dir)

    result = remote_host.run_command(start_command)
    u.print_remote_result(result)
    if result.stderr:
        raise RuntimeError("❌ Error starting observe-agent service")

    # Check that old version is running
    print("🔍 Verifying old version is running...")
    agent_status = _check_status_loop(remote_host, status_command)
    if not agent_status:
        u.die("❌ Old version failed to start")

    old_version = _get_agent_version(remote_host, version_command)
    print(f"✅ Old version {old_version} is running")

    # Now upgrade to the new version
    print("⬆️ Upgrading to new version...")

    # Get built dist. installation package path for machine
    current_dir = os.getcwd()
    dist_directory = os.path.abspath(os.path.join(current_dir, "..", "dist"))
    files = os.listdir(dist_directory)

    package_type = env_vars["machine_config"]["package_type"]
    architecture = env_vars["machine_config"]["architecture"]

    filename = None
    for file in files:
        if package_type in file and architecture in file and "windows" in file.lower():
            filename = file
            break

    if not filename:
        u.die(f"❌ No matching upgrade package found in {dist_directory}")

    full_path = os.path.join(dist_directory, filename)

    # Copy new package and install script
    remote_host.put_file(local_path=full_path, remote_path=home_dir)

    ps_install_script_path = os.path.join(current_script_dir, "install_windows.ps1")
    remote_host.put_file(local_path=ps_install_script_path, remote_path=home_dir)

    # Run upgrade installation
    home_dir_powershell = r"C:\Users\{}".format(env_vars["user"])
    upgrade_command = r".\install_windows.ps1 -local_installer {}\{}".format(
        home_dir_powershell, filename
    )

    result = remote_host.run_command(upgrade_command)
    u.print_remote_result(result)
    if result.stderr:
        raise RuntimeError("❌ Error upgrading observe-agent")

    # Check that service is still running after upgrade
    print("🔍 Verifying service is still running after upgrade...")
    agent_status = _check_status_loop(remote_host, status_command, num_retries=15)
    if not agent_status:
        u.die("❌ Service failed to remain running after upgrade")

    new_version = _get_agent_version(remote_host, version_command)
    print(f"✅ New version {new_version} is running")

    if old_version == new_version:
        print("⚠️ Warning: Version appears unchanged after upgrade")
    else:
        print(f"✅ Successfully upgraded from {old_version} to {new_version}")


@u.print_test_decorator
def run_test_docker(remote_host: u.Host, env_vars: dict) -> None:
    """
    Test to upgrade observe-agent in Docker from older version to current build

    Args:
        remote_host (Host): instance to ssh into
        env_vars (dict): environment variables passed into for testing
    """
    docker_prefix = u.get_docker_prefix(remote_host, True)

    print("🔄 Installing older version of observe-agent...")
    # Install old version using official Docker image
    old_install_command = f"sudo docker run -d --name observe-agent-old -v /etc/observe-agent:/etc/observe-agent observeinc/observe-agent:v2.5.0"
    result = remote_host.run_command(old_install_command)
    if result.stderr:
        u.print_remote_result(result)
        u.die("❌ Error installing older version of observe-agent")

    # Get old version info
    old_version_command = "sudo docker exec observe-agent-old ./observe-agent version"
    old_version = _get_agent_version(remote_host, old_version_command)
    print(f"✅ Old version {old_version} installed")

    # Stop old container
    remote_host.run_command("sudo docker stop observe-agent-old")
    remote_host.run_command("sudo docker rm observe-agent-old")

    # Now start with the new version being tested
    print("⬆️ Upgrading to new version...")
    start_command = "start"
    result = remote_host.run_command(docker_prefix + " " + start_command)
    if result.stderr:
        u.print_remote_result(result)
        u.die("❌ Error starting new observe-agent container")

    # Get new container ID and check status
    container_id = u.get_docker_container(remote_host)
    status_command = f"sudo docker exec {container_id} ./observe-agent status"
    version_command = f"sudo docker exec {container_id} ./observe-agent version"

    print("🔍 Verifying new version is running...")
    agent_status = _check_status_loop(remote_host, status_command)
    if not agent_status:
        u.die("❌ New version failed to start")

    new_version = _get_agent_version(remote_host, version_command)
    print(f"✅ Successfully upgraded from {old_version} to {new_version}")


@u.print_test_decorator
def run_test_linux(remote_host: u.Host, env_vars: dict) -> None:
    """
    Test to upgrade observe-agent on Linux from older version to current build

    Args:
        remote_host (Host): instance to ssh into
        env_vars (dict): environment variables passed into for testing
    """
    # Commands for Linux
    install_old_command = "curl -s -L https://github.com/observeinc/observe-agent/releases/download/v2.5.0/observe-agent_Linux_$(arch).tar.gz | sudo tar -xz -C /tmp && sudo /tmp/observe-agent/install_linux.sh"
    start_command = "sudo systemctl enable --now observe-agent"
    status_command = "observe-agent status"
    version_command = "observe-agent version"

    print("🔄 Installing older version of observe-agent...")
    result = remote_host.run_command(install_old_command)
    u.print_remote_result(result)
    if result.exited != 0:
        u.die("❌ Error installing older version of observe-agent")

    print("🚀 Starting observe-agent service...")
    result = remote_host.run_command(start_command)
    u.print_remote_result(result)

    # Check that old version is running
    print("🔍 Verifying old version is running...")
    agent_status = _check_status_loop(remote_host, status_command)
    if not agent_status:
        u.die("❌ Old version failed to start")

    old_version = _get_agent_version(remote_host, version_command)
    print(f"✅ Old version {old_version} is running")

    # Now upgrade to the new version
    print("⬆️ Upgrading to new version...")

    # Get built dist. installation package path for machine
    current_dir = os.getcwd()
    dist_directory = os.path.abspath(os.path.join(current_dir, "..", "dist"))
    files = os.listdir(dist_directory)

    package_type = env_vars["machine_config"]["package_type"]
    architecture = env_vars["machine_config"]["architecture"]
    distribution = env_vars["machine_config"]["distribution"]

    filename = None
    for file in files:
        if package_type in file and architecture in file:
            if "linux" in distribution.lower() and "linux" in file.lower():
                filename = file
                break

    if not filename:
        u.die(f"❌ No matching upgrade package found in {dist_directory}")

    full_path = os.path.join(dist_directory, filename)

    # Copy new package to remote host
    remote_host.put_file(local_path=full_path, remote_path="/tmp/observe-agent-new.tar.gz")

    # Extract and run upgrade installation
    upgrade_command = "sudo tar -xzf /tmp/observe-agent-new.tar.gz -C /tmp && sudo /tmp/observe-agent/install_linux.sh"
    result = remote_host.run_command(upgrade_command)
    u.print_remote_result(result)
    if result.exited != 0:
        u.die("❌ Error upgrading observe-agent")

    # Check that service is still running after upgrade
    print("🔍 Verifying service is still running after upgrade...")
    agent_status = _check_status_loop(remote_host, status_command, num_retries=15)
    if not agent_status:
        # If the agent isn't running, try to restart it and see what happens
        print("🔄 Attempting to restart service after upgrade...")
        restart_result = remote_host.run_command("sudo systemctl restart observe-agent")
        u.print_remote_result(restart_result)

        # Check again
        agent_status = _check_status_loop(remote_host, status_command)
        if not agent_status:
            u.die("❌ Service failed to remain running after upgrade")

    new_version = _get_agent_version(remote_host, version_command)
    print(f"✅ New version {new_version} is running")

    if old_version == new_version:
        print("⚠️ Warning: Version appears unchanged after upgrade")
    else:
        print(f"✅ Successfully upgraded from {old_version} to {new_version}")


if __name__ == "__main__":
    env_vars = u.get_env_vars()
    remote_host = u.Host(
        host_ip=env_vars["host"],
        username=env_vars["user"],
        key_file_path=env_vars["key_filename"],
        password=env_vars["password"],
    )

    # Test SSH Connection before starting test
    remote_host.test_conection(int(env_vars["machine_config"]["sleep"]))

    if (
        "redhat" in env_vars["machine_config"]["distribution"]
        or "debian" in env_vars["machine_config"]["distribution"]
    ):
        run_test_linux(remote_host, env_vars)
    elif "windows" in env_vars["machine_config"]["distribution"]:
        run_test_windows(remote_host, env_vars)
    elif "docker" in env_vars["machine_config"]["distribution"]:
        run_test_docker(remote_host, env_vars)