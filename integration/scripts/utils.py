from socket import error as socket_error

from fabric import Connection
from paramiko.ssh_exception import AuthenticationException, NoValidConnectionsError

import os
import sys
import re
import time 
import json 
import pprint 

def die(message: str) -> None:
    print(message, file=sys.stderr)
    sys.exit(1)

def mask_credentials(env_vars):
    masked_env_vars = env_vars.copy()
    if "password" in masked_env_vars and "password" is not None:
        masked_env_vars["password"] = '*' * 5
    if "observe_token" in masked_env_vars and "password" is not None:
        masked_env_vars["observe_token"] = '*' * 5
    return masked_env_vars

def get_env_vars(need_observe: bool = False) -> dict:
    

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
    password=os.environ.get("PASSWORD")
    machine_name=os.environ.get("MACHINE_NAME")
    machine_config_string=os.environ.get("MACHINE_CONFIG")
    observe_url=os.environ.get("OBSERVE_URL")
    observe_token=os.environ.get("OBSERVE_TOKEN")

    mask = os.getenv("MASK", "True").lower() not in ("false", "0", "f", "no", "n")


    if host is None:
        die("Error: HOST environment variable is not set. This should be an output variable from create_ec2 module")

    if user is None:
        die("Error: USER environment variable is not set. This should be an output variable from create_ec2 module")

    if key_filename is None:
        die("Error: KEY_FILENAME environment variable is not set. This should be an output variable from create_ec2 module")

    if password == 'None' and "WINDOWS" in machine_name:
        die("Error: Windows is specified but PASSWORD environment variable is not set. This should be an output variable from create_ec2 module")

    if machine_name is None:
        die("Error: MACHINE_NAME environment variable is not set. This should be an output variable from create_ec2 module")

    if machine_config_string is None:
        die("Error: MACHINE_CONFIG environment variable is not set. This should be an output variable from create_ec2 module")

    if observe_url is None and need_observe:
        die("Error: OBSERVE_URL environment variable is not set. This should be an output variable from setup_observe_variables module")
    if observe_token is None and need_observe:
        die("Error: OBSERVE_TOKEN environment variable is not set. This should be an output variable from setup_observe_variables module")

     # Split the string into key-value pairs
    pairs = machine_config_string.split(',')
    data = {}
    for pair in pairs:
        key, value = pair.split(':', 1)  #
        data[key] = value
    
    env_vars = {
        "host": host,
        "user": user,
        "key_filename": key_filename,
        "password": password,
        "machine_name": machine_name,
        "machine_config": data,
        "observe_url": observe_url,
        "observe_token": observe_token
    }

    # Mask sensitive vars before printing
    masked_env_vars = mask_credentials(env_vars)

    print("-"*30)
    if mask:
        print("Masking Enabled")
        print("Env vars set to: \n",  masked_env_vars )
    else:
        print("Masking Disabled")
        print("Env vars set to: \n",  env_vars )
    print("-"*30)

    return env_vars


def print_test_decorator(func):

    def wrapper(*args, **kwargs):
        print("*" * 30)
        print("Running Test:", func.__name__)
        result = func(*args, **kwargs)
        print("*" * 30)
        return result
    return wrapper

class ExampleException(Exception):  #We can put our custom exceptions here 
    pass


class Host(object):

    """Host class for SSH into EC2 instances 
    """
    def __init__(self, host_ip, username, key_file_path,password=None):
        self.host_ip = host_ip
        self.username = username
        self.key_file_path = key_file_path
        self.password=password

    def _get_connection(self) -> Connection:
        connect_kwargs = {'key_filename': self.key_file_path,                          
                          'password': self.password ,
                          'timeout': 60,                      
                          }
        return Connection(host=self.host_ip, user=self.username, port=22,
                          connect_kwargs=connect_kwargs)

    def run_command(self, command):
        try:
            with self._get_connection() as connection:
                print('Running `{0}` on {1}'.format(command, self.host_ip))
                result = connection.run(command, warn=True, hide=True)                
        except (socket_error, AuthenticationException) as exc:
            self._raise_authentication_err(exc)

        if result.failed:
            raise ExampleException(
                'The command `{0}` on host {1} failed with the error: '
                '{2}'.format(command, self.host_ip, str(result.stderr)))
        
        return result


    def put_file(self, local_path, remote_path) -> None:
        try:
            with self._get_connection() as connection:
                print('Copying {0} to {1} on host {2}'.format(
                    local_path, remote_path, self.host_ip))
                connection.put(local_path, remote_path)
        except (socket_error, AuthenticationException) as exc:
            self._raise_authentication_err(exc)

    def get_file(self, remote_path, local_path) -> None:
        try:
            with self._get_connection() as connection:
                print('Copying {0} to {1} from host {2}'.format(
                    remote_path, local_path, self.host_ip))
                connection.get(remote_path, local_path)
        except (socket_error, AuthenticationException) as exc:
            self._raise_authentication_err(exc)

    def _raise_authentication_err(self, exc):
        raise ExampleException(
            "SSH: could not connect to {host} "
            "(username: {user}, key: {key}): {exc}".format(
                host=self.host_ip, user=self.username,
                key=self.key_file_path, exc=exc))
    
    def test_conection(self, timeout=60):
        """Tests SSH connection to the host 

        Args:
            timeout (int, optional): how long to wait for the connection to be established. Defaults to 60.

        Raises:
            RuntimeError: SSH connection failures if the timeout is reached and no valid connection found
        """
        print("Testing SSH connection to host {} with timeout {}s".format(self.host_ip, timeout))
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

