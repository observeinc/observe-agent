#!/usr/bin/env python3
import utils as u
import os
import difflib

AGENT_CONFIG_FILE_NAME = "full-agent-config.yaml"


def upload_agent_config(remote_host: u.Host, remote_path: str) -> None:
    current_script_dir = os.path.dirname(os.path.abspath(__file__))
    # Upload the full observe-agent config to remote host
    local_path = os.path.abspath(
        os.path.join(current_script_dir, "snapshots", AGENT_CONFIG_FILE_NAME)
    )
    print(f"Path to '{AGENT_CONFIG_FILE_NAME}' file: {local_path}")
    remote_host.put_file(local_path=local_path, remote_path=remote_path)


def do_snapshot_test(
    remote_host: u.Host, remote_path: str, config_command: str, snapshot_file: str
) -> None:
    # Copy the agent config to the remote host
    upload_agent_config(remote_host, remote_path)
    result = remote_host.run_command(config_command)
    if result.exited != 0 or result.stderr:
        u.print_remote_result(result)
        raise ValueError(f"❌ Error in rendering config for {snapshot_file}")
    rendered_config = result.stdout
    current_script_dir = os.path.dirname(os.path.abspath(__file__))
    with open(os.path.join(current_script_dir, "snapshots", snapshot_file), "r") as f:
        expected_config = f.read()
    if rendered_config == expected_config:
        print(f" ✅ Config match for {snapshot_file}")
        return
    diff = difflib.Differ().compare(
        expected_config.splitlines(), rendered_config.splitlines()
    )
    print(f"Diff expected vs actual for {snapshot_file}:")
    print("\n".join(diff))
    raise ValueError(f"❌ Config mismatch for {snapshot_file}")


@u.print_test_decorator
def run_test_windows(remote_host: u.Host, env_vars: dict) -> None:
    user = env_vars["user"]
    remote_path = f"/C:/Users/{user}"  # for user in sftp
    config_command = f"Set-Location \"C:\\Program Files\\Observe\\observe-agent\"; ./observe-agent --observe-config C:\\Users\\{user}\\{AGENT_CONFIG_FILE_NAME} config --render-otel"
    snapshot_file = "windows.yaml"
    do_snapshot_test(remote_host, remote_path, config_command, snapshot_file)


@u.print_test_decorator
def run_test_docker(remote_host: u.Host, env_vars: dict) -> None:
    docker_prefix = u.get_docker_prefix(
        remote_host,
        False,
        extra_args=f"--mount type=bind,source=$(pwd)/{AGENT_CONFIG_FILE_NAME},target=/etc/observe-agent/{AGENT_CONFIG_FILE_NAME}",
    )
    remote_path = "/home/" + env_vars["user"]
    config_command = f"{docker_prefix} --observe-config /etc/observe-agent/{AGENT_CONFIG_FILE_NAME} config --render-otel"
    snapshot_file = "docker.yaml"
    do_snapshot_test(remote_host, remote_path, config_command, snapshot_file)


@u.print_test_decorator
def run_test_linux(remote_host: u.Host, env_vars: dict) -> None:
    remote_path = "/home/" + env_vars["user"]
    config_command = f"observe-agent --observe-config {remote_path}/{AGENT_CONFIG_FILE_NAME} config --render-otel"
    snapshot_file = "linux.yaml"
    do_snapshot_test(remote_host, remote_path, config_command, snapshot_file)


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
