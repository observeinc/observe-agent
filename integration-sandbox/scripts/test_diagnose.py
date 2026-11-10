#!/usr/bin/env python3
from fabric import Result

import re
import utils as u


def _check_diagnose_result(result: Result) -> bool:
    passed = re.search(r"All \d+ checks passed", result.stdout) is not None
    if passed:
        print(" ✅ observe-agent -> observe validation passed! ")
    else:
        u.print_remote_result(result)
        raise ValueError(
            f"❌ Failed: observe-agent -> observe validation (regex on diagnose output did not match)"
        )


@u.print_test_decorator
def run_test_windows(remote_host: u.Host, env_vars: dict) -> None:
    diagnose_command = r'Set-Location "C:\Program Files\Observe\observe-agent"; ./observe-agent diagnose'

    # Check diagnose command
    result = remote_host.run_command(diagnose_command)
    _check_diagnose_result(result)


@u.print_test_decorator
def run_test_docker(remote_host: u.Host, env_vars: dict) -> None:
    container_id = u.get_docker_container(remote_host)
    exec_prefix = f"sudo docker exec {container_id} ./observe-agent"
    diagnose_command = exec_prefix + " diagnose"

    # Check diagnose command
    result = remote_host.run_command(diagnose_command)
    _check_diagnose_result(result)


@u.print_test_decorator
def run_test_linux(remote_host: u.Host, env_vars: dict) -> None:
    diagnose_command = "observe-agent diagnose"

    # Check diagnose command
    result = remote_host.run_command(diagnose_command)
    _check_diagnose_result(result)


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
