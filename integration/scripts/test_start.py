#!/usr/bin/env python3
import os
import sys
import re
import time
import pprint
import utils as u
from collections import defaultdict


def _check_status_loop(
    remote_host: u.Host, start_timeout: int, status_command: str
) -> bool:
    """Run Check Status Command in a loop to wait for observe-agent to start

    Args:
        remote_host (Host): instance to ssh into
        start_timeout (int): timeout in seconds to wait for agent to start
        status_command (str): windows/linux status command to run

    Returns:
        bool: agent_status
    """

    agent_status = False
    for _ in range(start_timeout):
        metrics_dict = defaultdict(list)
        try:
            result = remote_host.run_command(status_command)
        except Exception as e:
            print("Ignoring exception: ", e)
            time.sleep(1)
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
                _ + 1, start_timeout
            )
        )
        time.sleep(1)
    return agent_status


@u.print_test_decorator
def run_test_windows(remote_host: u.Host, env_vars: dict) -> None:
    """
    Test to check if observe-agent is running correctly

    Args:
        remote_host (Host): instance to ssh into
        env_vars (dict): environment variables passed into for testing

    """

    # status
    start_command = r".\start_agent_windows.ps1"
    status_command = r'Get-Service ObserveAgent;Set-Location "${Env:Programfiles}\Observe\observe-agent"; ./observe-agent status'
    start_timeout = 30  # how long to wait for observe-agent to start

    # Get windows home dir paths for consistency
    home_dir = r"/C:/Users/{}".format(env_vars["user"])  # for user in sftp

    # Find start agent script path
    current_script_dir = os.path.dirname(os.path.abspath(__file__))
    ps_installation_script_path = os.path.join(
        current_script_dir, "start_agent_windows.ps1"
    )

    # Copy start_agent powershell installation script to remote host home dir
    remote_host.put_file(
        local_path=ps_installation_script_path, remote_path=home_dir
    )  # Eg: sftp to /C:/Users/Adminstrator/install_windows.ps1
    # Run start_agent script
    result = remote_host.run_command(start_command)
    print(result)

    if (
        result.stderr
    ):  # Powershell script failure does not cause command failure as the installation command succeeds so we need to check the stderr
        raise RuntimeError("❌ Error in start_agent_windows.ps1 powershell script")

    # Check Agent Status
    agent_status = _check_status_loop(remote_host, start_timeout, status_command)
    if not agent_status:
        u.die("❌ Error in Observe Agent Status Test ")


@u.print_test_decorator
def run_test_docker(remote_host: u.Host, env_vars: dict) -> None:
    docker_prefix = u.get_docker_prefix(remote_host, True)
    start_command = "start"
    start_timeout = 30  # how long to wait for observe-agent to start

    # Start Observe Agent
    result = remote_host.run_command(docker_prefix + " " + start_command)
    if result.stderr:
        u.die("❌ Error starting observe-agent container")
    else:
        print("✅ Observe Agent started successfully: " + result.stdout)

    # Get Observe Agent Container ID
    container_id = u.get_docker_container(remote_host)
    status_command = f"sudo docker exec {container_id} ./observe-agent status"

    # Check Agent Status
    agent_status = _check_status_loop(remote_host, start_timeout, status_command)
    if not agent_status:
        u.die("❌ Error in Observe Agent Status Test ")


@u.print_test_decorator
def run_test_linux(remote_host: u.Host, env_vars: dict) -> None:
    """
    Test to check if observe-agent is running correctly

    Args:
        remote_host (Host): instance to ssh into
        env_vars (dict): environment variables passed into for testing

    """

    start_command = "sudo systemctl enable --now observe-agent"
    status_command = "observe-agent status"
    start_timeout = 30  # how long to wait for observe-agent to start

    # Start Observe Agent
    remote_host.run_command(start_command)

    # Check Agent Status
    agent_status = _check_status_loop(remote_host, start_timeout, status_command)
    if not agent_status:
        u.die("❌ Error in Observe Agent Status Test ")


if __name__ == "__main__":

    env_vars = u.get_env_vars()
    remote_host = u.Host(
        host_ip=env_vars["host"],
        username=env_vars["user"],
        key_file_path=env_vars["key_filename"],
        password=env_vars["password"],
    )

    # Test SSH Connection before starting test of interest
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
