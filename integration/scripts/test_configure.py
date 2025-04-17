#!/usr/bin/env python3
import utils as u


def init_config_command(env_vars: dict) -> str:
    return "init-config --token {} --observe_url {} --cloud_resource_detectors ec2 --resource_attributes deployment.environment=test".format(
        env_vars["observe_token"], env_vars["observe_url"]
    )


def run_init_config_common(remote_host: u.Host, init_command: str) -> None:
    # Print the config to be used first
    result = remote_host.run_command(init_command + " --print")
    print("Setting agent config:\n{}\n".format("=" * 21))
    u.print_remote_result(result)
    if result.exited != 0 or result.stderr:
        raise ValueError("❌ Error in init-config print")

    # Set up correct config with observe url and token
    result = remote_host.run_command(init_command)
    if result.exited != 0 or result.stderr:
        u.print_remote_result(result)
        raise ValueError("❌ Error in init-config")


@u.print_test_decorator
def run_test_windows(remote_host: u.Host, env_vars: dict) -> None:
    init_command = (
        r'Set-Location "C:\Program Files\Observe\observe-agent"; ./observe-agent '
        + init_config_command(env_vars)
    )
    run_init_config_common(remote_host, init_command)


@u.print_test_decorator
def run_test_docker(remote_host: u.Host, env_vars: dict) -> None:
    docker_prefix = u.get_docker_prefix(remote_host, False)
    init_command = docker_prefix + " " + init_config_command(env_vars)
    run_init_config_common(remote_host, init_command)


@u.print_test_decorator
def run_test_linux(remote_host: u.Host, env_vars: dict) -> None:
    init_command = "sudo observe-agent " + init_config_command(env_vars)
    run_init_config_common(remote_host, init_command)


if __name__ == "__main__":

    env_vars = u.get_env_vars(need_observe=True)
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
