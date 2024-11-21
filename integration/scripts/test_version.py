#!/usr/bin/env python3
import os
import re
import utils as u


def _extract_version_config(result: any) -> tuple:
    """Extract version name and config file from ssh result output

    Args:
        result (any): ssh result output

    Returns:
        tuple: config_file, version of the installed observe-agent package
    """

    # Split the output by newlines and extract everything after the colon
    version_match = re.search(r"version: (.*)(?:\n|$)", result.stdout)
    if version_match is not None:
        version = version_match.group(1).strip()
    else:
        raise ValueError(
            f"❌ Failed: observe-agent version output did not match regex. Output: {result.stdout}"
        )

    config_match = re.search(r"config file: (.*)(?:\n|$)", result.stdout)
    if config_match is not None:
        config_file = config_match.group(1).strip()
    else:
        raise ValueError(
            f"❌ Failed: observe-agent version output did not match regex. Output: {result.stdout}"
        )
    print(f"Version: {version}, Config File: {config_file}")
    return config_file, version


@u.print_test_decorator
def run_test_windows(remote_host: u.Host, env_vars: dict) -> None:
    """
    Test to validate observe-agent version and config file loaded is correct

    Args:
        remote_host (Host): instance to ssh into
        env_vars (dict): environment variables passed into for testing

    Raises:
        ValueError: if version or config file is invalid
    """

    config_file_windows = (
        "C:\\Program Files\\Observe\\observe-agent\\observe-agent.yaml"
    )
    # Can match 0.2.2-SNAPSHOT-b6e1491 or 0.2.2
    version_pattern = re.compile(r"^\d+\.\d+\.\d+(-[A-Za-z0-9-]+)?$")

    result = remote_host.run_command(
        'Set-Location "${Env:Programfiles}\\Observe\\observe-agent"; ./observe-agent version'
    )
    config_file, version = _extract_version_config(result)

    if config_file != config_file_windows:
        raise ValueError(f" ❌ Invalid config file: {config_file}")
    if not version_pattern.match(version):
        raise ValueError(f" ❌ Invalid version: {version}")

    print(" ✅ Verified version and config file succesfully! ")

    pass


@u.print_test_decorator
def run_test_docker(remote_host: u.Host, env_vars: dict) -> None:
    u.upload_default_docker_config(env_vars, remote_host)
    docker_prefix = u.get_docker_prefix(remote_host, False)
    config_file_linux = "/etc/observe-agent/observe-agent.yaml"
    version_pattern = re.compile(r"^\d+\.\d+\.\d+(-[A-Za-z0-9-]+)?$")

    # Run command to get version & config-file info
    result = remote_host.run_command(docker_prefix + " version")
    config_file, version = _extract_version_config(result)

    if config_file != config_file_linux:
        raise ValueError(f" ❌ Invalid config file: {config_file}")
    if not version_pattern.match(version):
        raise ValueError(f" ❌ Invalid version: {version}")

    print(" ✅ Verified version and config file succesfully! ")


@u.print_test_decorator
def run_test_linux(remote_host: u.Host, env_vars: dict) -> None:
    """
    Test to validate observe-agent version and config file loaded is correct

    Args:
        remote_host (Host): instance to ssh into
        env_vars (dict): environment variables passed into for testing

    Raises:
        ValueError: if version or config file is invalid
    """
    config_file_linux = "/etc/observe-agent/observe-agent.yaml"
    # Can match 0.2.2-SNAPSHOT-b6e1491 or 0.2.2
    version_pattern = re.compile(r"^\d+\.\d+\.\d+(-[A-Za-z0-9-]+)?$")

    result = remote_host.run_command("observe-agent version")
    config_file, version = _extract_version_config(result)

    if config_file != config_file_linux:
        raise ValueError(f" ❌ Invalid config file: {config_file}")
    if not version_pattern.match(version):
        raise ValueError(f" ❌ Invalid version: {version}")

    print(" ✅ Verified version and config file succesfully! ")


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
