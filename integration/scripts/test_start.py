#!/usr/bin/env python3
import os
import sys
import re
import time
import pprint
import utils as u


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
    u.print_remote_result(result)

    if (
        result.stderr
    ):  # Powershell script failure does not cause command failure as the installation command succeeds so we need to check the stderr
        raise RuntimeError("❌ Error in start_agent_windows.ps1 powershell script")

    # Check Agent Status
    agent_status = u.check_status_loop(remote_host, status_command)
    if not agent_status:
        u.die("❌ Error in Observe Agent Status Test ")


@u.print_test_decorator
def run_test_docker(remote_host: u.Host, env_vars: dict) -> None:
    docker_prefix = u.get_docker_prefix(remote_host, True)
    start_command = "start"

    # Start Observe Agent
    result = remote_host.run_command(docker_prefix + " " + start_command)
    if result.stderr:
        u.print_remote_result(result)
        u.die("❌ Error starting observe-agent container")
    else:
        print("✅ Observe Agent started successfully: " + result.stdout)

    # Get Observe Agent Container ID
    container_id = u.get_docker_container(remote_host)
    status_command = f"sudo docker exec {container_id} ./observe-agent status"

    # Check Agent Status
    agent_status = u.check_status_loop(remote_host, status_command)
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

    # Start Observe Agent
    result = remote_host.run_command(start_command)
    u.print_remote_result(result)

    # Check Agent Status
    agent_status = u.check_status_loop(remote_host, status_command)
    if not agent_status:
        # If the agent never started up, try running start to see what the error is. Use unsafe because we expect a non-zero exit code.
        u.print_remote_result(
            remote_host.run_command_unsafe("timeout 10s sudo observe-agent start")
        )
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
