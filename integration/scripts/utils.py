from socket import error as socket_error

from fabric import Connection
from paramiko.ssh_exception import AuthenticationException, NoValidConnectionsError

import os
import sys
import re
import time 

def die(message):
    print(message, file=sys.stderr)
    sys.exit(1)


def check_env_vars():
    host = os.environ.get("HOST")
    user = os.environ.get("USER")
    key_filename = os.environ.get("KEY_FILENAME")
    machine_name=os.environ.get("MACHINE_NAME")

    if host is None:
        die("Error: HOST environment variable is not set.")

    if user is None:
        die("Error: USER environment variable is not set.")

    if key_filename is None:
        die("Error: KEY_FILENAME environment variable is not set.")

    if machine_name is None:
        die("Error: MACHINE_NAME environment variable is not set.")

    return host, user, key_filename, machine_name

class ExampleException(Exception):  #We can put our custom exceptions here 
    pass


class Host(object):
    def __init__(self,
                 host_ip,
                 username,
                 key_file_path):
        self.host_ip = host_ip
        self.username = username
        self.key_file_path = key_file_path

    def _get_connection(self):
        connect_kwargs = {'key_filename': self.key_file_path,
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


    def put_file(self, local_path, remote_path):
        try:
            with self._get_connection() as connection:
                print('Copying {0} to {1} on host {2}'.format(
                    local_path, remote_path, self.host_ip))
                connection.put(local_path, remote_path)
        except (socket_error, AuthenticationException) as exc:
            self._raise_authentication_err(exc)

    def get_file(self, remote_path, local_path):
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
    
    def test_conection(self, timeout=30):

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
