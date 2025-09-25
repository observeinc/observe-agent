from typing import Any, Dict
from socket import error as socket_error
from collections import defaultdict

from fabric import Connection, Result
from paramiko.ssh_exception import AuthenticationException, NoValidConnectionsError

import os
import re
import sys
import time


def die(message: str) -> None:
    print(message, file=sys.stderr)
    sys.exit(1)


def print_remote_result(result: Result) -> None:
    print(str(result))


def mask_credentials(env_vars: Dict[str, Any]) -> Dict[str, Any]:
    masked_env_vars = env_vars.copy()
    # Only mask if vars exist
    if (
        masked_env_vars["password"]
        and masked_env_vars["password"] is not None
        and masked_env_vars["password"] != "None"
    ):
        masked_env_vars["password"] = "*" * 5
    if (
        masked_env_vars["observe_token"]
        and masked_env_vars["observe_token"] is not None
        and masked_env_vars["observe_token"] != "None"
    ):
        masked_env_vars["observe_token"] = "*" * 5
    return masked_env_vars


def get_env_vars(need_observe: bool = False) -> Dict[str, Any]:
    """Gets environmental variables from OS and returns a dict of env_vars

    Args:
        need_observe (bool, optional): whether or not to require observe url/token variables.
          Defaults to False.

    Returns:
        _type_: dict of environment variables
    """
    host = os.environ.get("HOST")
    user = os.environ.get("USER")
    key_filename = os.environ.get("KEY_FILENAME")
    password = os.environ.get("PASSWORD")
    machine_name = os.environ.get("MACHINE_NAME")
    machine_config_string = os.environ.get("MACHINE_CONFIG")
    observe_url = os.environ.get("OBSERVE_URL")
    observe_token = os.environ.get("OBSERVE_TOKEN")

    mask = os.getenv("MASK", "True").lower() not in ("false", "0", "f", "no", "n")

    if host is None:
        die(
            "Error: HOST environment variable is not set. This should be an output variable from create_ec2 module"
        )

    if user is None:
        die(
            "Error: USER environment variable is not set. This should be an output variable from create_ec2 module"
        )

    if key_filename is None:
        die(
            "Error: KEY_FILENAME environment variable is not set. This should be an output variable from create_ec2 module"
        )

    if (password == "None" or password is None) and "WINDOWS" in machine_name:
        die(
            "Error: Windows is specified but PASSWORD environment variable is not set. This should be an output variable from create_ec2 module"
        )

    if machine_name is None:
        die(
            "Error: MACHINE_NAME environment variable is not set. This should be an output variable from create_ec2 module"
        )

    if machine_config_string is None:
        die(
            "Error: MACHINE_CONFIG environment variable is not set. This should be an output variable from create_ec2 module"
        )

    if observe_url is None and need_observe:
        die(
            "Error: OBSERVE_URL environment variable is not set. This should be an output variable from setup_observe_variables module"
        )
    if observe_token is None and need_observe:
        die(
            "Error: OBSERVE_TOKEN environment variable is not set. This should be an output variable from setup_observe_variables module"
        )

    # Split the string into key-value pairs
    pairs = machine_config_string.split(",")
    data = {}
    for pair in pairs:
        key, value = pair.split(":", 1)  #
        data[key] = value

    env_vars = {
        "host": host,
        "user": user,
        "key_filename": key_filename,
        "password": password,
        "machine_name": machine_name,
        "machine_config": data,
        "observe_url": observe_url,
        "observe_token": observe_token,
    }

    # Mask sensitive vars before printing
    masked_env_vars = mask_credentials(env_vars)

    print("-" * 30)
    if mask:
        print("Masking Enabled")
        print("Env vars set to: \n", masked_env_vars)
    else:
        print("Masking Disabled")
        print("Env vars set to: \n", env_vars)
    print("-" * 30)

    return env_vars


def print_test_decorator(func):

    def wrapper(*args, **kwargs):
        print("*" * 30)
        print("Running Test:", func.__name__)
        result = func(*args, **kwargs)
        print("*" * 30)
        return result

    return wrapper


class ExampleException(Exception):  # We can put our custom exceptions here
    pass


class Host(object):
    """Host class for SSH into EC2 instances"""

    def __init__(self, host_ip, username, key_file_path, password=None):
        self.host_ip = host_ip
        self.username = username
        self.key_file_path = key_file_path
        self.password = password

    def _get_connection(self) -> Connection:
        connect_kwargs = {
            "key_filename": self.key_file_path,
            "password": self.password,
            "timeout": 60,
        }
        return Connection(
            host=self.host_ip,
            user=self.username,
            port=22,
            connect_kwargs=connect_kwargs,
        )

    def run_command(self, command) -> Result:
        result = self.run_command_unsafe(command)

        if result.failed:
            raise ExampleException(
                "The command `{0}` on host {1} failed with the error: "
                "{2}\n\nCommand output: {3}".format(
                    command,
                    self.host_ip,
                    str(result.stderr) or "<empty>",
                    str(result.stdout) or "<empty>",
                )
            )

        return result

    def run_command_unsafe(self, command) -> Result:
        try:
            with self._get_connection() as connection:
                print("Running `{0}` on {1}".format(command, self.host_ip))
                result = connection.run(command, warn=True, hide=True)
        except (socket_error, AuthenticationException) as exc:
            self._raise_authentication_err(exc)

        return result

    def put_file(self, local_path, remote_path) -> None:
        try:
            with self._get_connection() as connection:
                print(
                    "Copying {0} to {1} on host {2}".format(
                        local_path, remote_path, self.host_ip
                    )
                )
                connection.put(local_path, remote_path)
        except (socket_error, AuthenticationException) as exc:
            self._raise_authentication_err(exc)

    def get_file(self, remote_path, local_path) -> None:
        try:
            with self._get_connection() as connection:
                print(
                    "Copying {0} to {1} from host {2}".format(
                        remote_path, local_path, self.host_ip
                    )
                )
                connection.get(remote_path, local_path)
        except (socket_error, AuthenticationException) as exc:
            self._raise_authentication_err(exc)

    def _raise_authentication_err(self, exc) -> None:
        raise ExampleException(
            "SSH: could not connect to {host} "
            "(username: {user}, key: {key}): {exc}".format(
                host=self.host_ip, user=self.username, key=self.key_file_path, exc=exc
            )
        )

    def test_conection(self, timeout=60) -> None:
        """Tests SSH connection to the host

        Args:
            timeout (int, optional): how long to wait for the connection to be established. Defaults to 60.

        Raises:
            RuntimeError: SSH connection failures if the timeout is reached and no valid connection found
        """
        print(
            "Testing SSH connection to host {} with timeout {}s".format(
                self.host_ip, timeout
            )
        )
        for _ in range(timeout):
            connection = self._get_connection()
            try:
                connection.open()
                print("✅ SSH connection successful")
                connection.close()
                return
            except (socket_error, NoValidConnectionsError) as exc:
                print(f"❌ SSH connection failed: {exc}")
            time.sleep(1)
        raise RuntimeError(" ❌ The SSH connection failed")


def check_status_loop(
    remote_host: "Host", status_command: str, num_retries: int = 10, sleep_seconds: float = 1.5
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


def get_agent_version(remote_host: Host, version_command: str) -> str:
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


def get_docker_image(remote_host: Host) -> str:
    result = remote_host.run_command(
        'sudo docker images --format "{{.Repository}}:{{.Tag}}"'
    )
    images = [line.strip() for line in result.stdout.splitlines() if "SNAPSHOT" in line]
    if len(images) != 1:
        die("❌ Error in finding observe-agent image\n" + str(result))

    return images[0]


def get_docker_prefix(remote_host: Host, detach: bool, extra_args: str = "") -> str:
    image = get_docker_image(remote_host)
    return f'sudo docker run {"-d --restart on-failure" if detach else ""} \
        --mount type=bind,source=/proc,target=/hostfs/proc,readonly \
        --mount type=bind,source=/snap,target=/hostfs/snap,readonly \
        --mount type=bind,source=/boot,target=/hostfs/boot,readonly \
        --mount type=bind,source=/var/lib,target=/hostfs/var/lib,readonly \
        --mount type=bind,source=/var/log,target=/hostfs/var/log,readonly \
        --mount type=bind,source=/var/lib/docker/containers,target=/var/lib/docker/containers,readonly \
        --mount type=bind,source=$(pwd)/observe-agent.yaml,target=/etc/observe-agent/observe-agent.yaml \
        {extra_args} \
        --pid host {image}'


def upload_default_docker_config(env_vars: dict, remote_host: Host) -> None:
    home_dir = "/home/{}".format(env_vars["user"])
    # Upload default observe-agent.yaml to remote host home dir
    # mount via $(pwd)/observe-agent.yaml,target=/etc/observe-agent/observe-agent.yaml
    observe_agent_file_path = os.path.abspath(
        os.path.join(os.getcwd(), "..", "packaging/linux/config/observe-agent.yaml")
    )
    print(f"Path to 'observe-agent.yaml' file: {observe_agent_file_path }")
    remote_host.put_file(local_path=observe_agent_file_path, remote_path=home_dir)


def get_docker_container(remote_host: Host) -> str:
    get_container_command = 'sudo docker ps --filter "status=running" --format "{{.ID}} {{.Image}} {{.CreatedAt}}"'
    result = remote_host.run_command(get_container_command)
    running = [
        line.strip() for line in result.stdout.splitlines() if "SNAPSHOT" in line
    ]
    if len(running) == 0:
        # No container matched our filter. Get logs from all containers to help debug.
        result = remote_host.run_command('sudo docker ps --format "{{.ID}}"')
        if result.stdout != "":
            container_ids = result.stdout.splitlines()
            for container_id in container_ids:
                print(
                    "Logs for container {}:".format(container_id),
                    file=sys.stderr,
                )
                result = remote_host.run_command(
                    "sudo docker logs {}".format(container_id)
                )
                print_remote_result(result)
        else:
            print_remote_result(result)
        die(
            "❌ Error in finding observe-agent container; command output:\n{}\ncommand error:\n{}".format(
                result.stdout or "<empty>",
                result.stderr or "<empty>",
            )
        )
        return ""
    if len(running) > 1:
        die(
            "❌ Error in finding observe-agent container, too many snapshots running:\n"
            + result.stdout
        )
    # Only one snapshot running; return the ID from the first line.
    return running[0].split()[0]
