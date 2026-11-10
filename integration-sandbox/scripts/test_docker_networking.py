#!/usr/bin/env python3
"""
Test that the agent's listener endpoints are reachable from outside the container.

This validates the 0.0.0.0 bind address in the Docker-packaged config by running
curl from a separate container on the same Docker network.
"""
import utils as u

DOCKER_NETWORK = "observe-test-net"


def _curl_from_container(remote_host: u.Host, network: str, url: str) -> None:
    cmd = (
        f"sudo docker run --rm --network {network} curlimages/curl"
        f" curl -sf --max-time 5 {url}"
    )
    result = remote_host.run_command(cmd)
    u.print_remote_result(result)


@u.print_test_decorator
def run_test_docker(remote_host: u.Host, env_vars: dict) -> None:
    container_id = u.get_docker_container(remote_host)

    remote_host.run_command(f"sudo docker network create {DOCKER_NETWORK}")
    remote_host.run_command(
        f"sudo docker network connect {DOCKER_NETWORK} {container_id}"
    )
    try:
        print("Testing health check endpoint (port 13133)...")
        _curl_from_container(
            remote_host, DOCKER_NETWORK, f"http://{container_id}:13133/status"
        )
        print("  ✅ Health check reachable from outside the container")

        print("Testing internal telemetry endpoint (port 8888)...")
        _curl_from_container(
            remote_host, DOCKER_NETWORK, f"http://{container_id}:8888/metrics"
        )
        print("  ✅ Internal telemetry reachable from outside the container")

        print("Testing OTLP HTTP receiver (port 4318)...")
        cmd = (
            f"sudo docker run --rm --network {DOCKER_NETWORK} curlimages/curl"
            f" curl -sf --max-time 5 -X POST"
            f" http://{container_id}:4318/v1/traces"
            f" -H 'Content-Type: application/json'"
            f" -d '{{\"resourceSpans\":[]}}'"
        )
        result = remote_host.run_command(cmd)
        u.print_remote_result(result)
        print("  ✅ OTLP HTTP receiver reachable from outside the container")

    finally:
        remote_host.run_command_unsafe(
            f"sudo docker network disconnect {DOCKER_NETWORK} {container_id}"
        )
        remote_host.run_command_unsafe(f"sudo docker network rm {DOCKER_NETWORK}")


if __name__ == "__main__":

    env_vars = u.get_env_vars()
    remote_host = u.Host(
        host_ip=env_vars["host"],
        username=env_vars["user"],
        key_file_path=env_vars["key_filename"],
        password=env_vars["password"],
    )

    remote_host.test_conection(int(env_vars["machine_config"]["sleep"]))

    if "docker" in env_vars["machine_config"]["distribution"]:
        run_test_docker(remote_host, env_vars)
    else:
        print("Skipping: test only applies to docker distribution")
