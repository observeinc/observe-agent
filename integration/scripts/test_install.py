#!/usr/bin/env python3


import os
import sys
import re
import time
import inspect
import utils as u


def _get_installation_package(env_vars: dict) -> tuple:
    """Returns the full path and filename to the built distribution package

    Args:
        env_vars (dict):environment variables passed into for testing

    Returns:
        tuple: filename and full path to the built distribution package
        Examples:
          filename: observe-agent_Windows_x86_64.zip
          full_path: /Users/nikhil.dua/Documents/observe-repos/observe-agent/dist/observe-agent_Windows_x86_64.zip

    """
    current_dir = os.getcwd()
    dist_directory = os.path.abspath(os.path.join(current_dir, "..", "dist"))
    print(f"Path to 'dist' directory: {dist_directory}")

    # List files in the directory
    files = os.listdir(dist_directory)

    # Search criteria
    package_type = env_vars["machine_config"]["package_type"]
    architecture = env_vars["machine_config"]["architecture"]
    distribution = env_vars["machine_config"]["distribution"]

    print(
        f"Looking for installation package '{package_type}' and architecture '{architecture}'"
    )

    # Iterate through files and find matches
    for filename in files:
        if package_type in filename and architecture in filename:
            # We can make this more general if need be.
            if "windows" in distribution and "windows" not in filename.lower():
                continue
            full_path = os.path.join(dist_directory, filename)
            print(f"Found matching file {filename} at: {full_path}")
            return filename, full_path
    u.die(
        f"❌ No matching file found for {distribution},{architecture},{package_type} in {dist_directory}: {', '.join(files)}"
    )


@u.print_test_decorator
def run_test_windows(remote_host: u.Host, env_vars: dict) -> None:
    """
    Test to install local observe-agent on a windows ec2 instance and validate command ran successfully

    Args:
        remote_host (Host): instance to ssh into
        env_vars (dict): environment variables passed into for testing

    Raises:
        RuntimeError: Installation error in powershell script
    """
    # Get built dist. installation package path for machine
    filename, full_path = _get_installation_package(env_vars)

    # Set windows home dir paths for consistency
    home_dir = r"/C:/Users/{}".format(env_vars["user"])  # for user in sftp
    home_dir_powershell = r"C:\Users\{}".format(
        env_vars["user"]
    )  # for use in powershell script

    # Find agent installation script path
    current_script_dir = os.path.dirname(os.path.abspath(__file__))
    ps_installation_script_path = os.path.join(
        current_script_dir, "install_windows.ps1"
    )

    # Copy built distribution package to remote host home dir
    remote_host.put_file(
        local_path=full_path, remote_path=home_dir
    )  # Eg: sftp to /C:/Users/Adminstrator/observe-agent_Windows_x86_64.zip

    # Copy observe-agent powershell installation script to remote host home dir
    remote_host.put_file(
        local_path=ps_installation_script_path, remote_path=home_dir
    )  # Eg: sftp to /C:/Users/Adminstrator/install_windows.ps1

    # Run install script and pass in distribution package path
    # Eg: .\install_windows.ps1 -local_installer C:\Users\Adminstrator\observe-agent_Windows_x86_64.zip
    # observe-agent gets installed to C:\Program Files\observe-agent on ec2 machine
    result = remote_host.run_command(
        r".\install_windows.ps1 -local_installer {}\{}".format(
            home_dir_powershell, filename
        )
    )
    print(result)

    if (
        result.stderr
    ):  # Powershell script failure does not cause command failure as the installation command succeeds so we need to check the stderr
        raise RuntimeError(
            "❌ Installation error in install_windows.ps1 powershell script"
        )
    else:
        print("✅ Installation test passed")


@u.print_test_decorator
def run_test_docker(remote_host: u.Host, env_vars: dict) -> None:

    filename, full_path = _get_installation_package(env_vars)
    home_dir = "/home/{}".format(env_vars["user"])

    remote_host.put_file(full_path, home_dir)
    result = remote_host.run_command("sudo docker load --input {}".format(filename))
    if result.stderr:
        print(result)
        raise RuntimeError("❌ Installation error in docker load")
    else:
        print("✅ Installation test passed")


@u.print_test_decorator
def run_test_linux(remote_host: u.Host, env_vars: dict):
    """
    Test to install local observe-agent on a linux ec2 instance and validate command ran successfully

    Args:
        remote_host (Host): instance to ssh into
        env_vars (dict): environment variables passed into for testing

    Raises:
        RuntimeError: Unknown distribution type passed
    """
    filename, full_path = _get_installation_package(env_vars)
    home_dir = "/home/{}".format(env_vars["user"])

    remote_host.put_file(full_path, home_dir)
    if "redhat" in env_vars["machine_config"]["distribution"]:
        result = remote_host.run_command(
            "cd ~ && sudo yum localinstall {} -y".format(filename)
        )
    elif "debian" in env_vars["machine_config"]["distribution"]:
        result = remote_host.run_command("cd ~ && sudo dpkg -i {}".format(filename))
    else:
        raise RuntimeError("❌ Unknown distribution type")

    print(result)
    print("✅ Installation test passed")


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
