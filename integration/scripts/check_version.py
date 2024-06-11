#!/usr/bin/env python3
import os
import sys
import re
import time 
from utils import Host, check_env_vars, die



def run_test_linux(remote_host: Host) -> None:    

    config_file_linux = '/etc/observe-agent/observe-agent.yaml'
    version_pattern = re.compile(r'^\d+\.\d+\.\d+$')
    connection_timeout = 30

    #Test SSH Connection 
    remote_host.test_conection(connection_timeout)

    result = remote_host.run_command('observe-agent version')    
    # Split the output by newlines and extract everything after the colon
    for line in result.stdout.splitlines():      
        if ":" in line:
            _, version = line.split(":", 1)
            version = version.strip()  # Remove leading/trailing whitespace
        print(f"Version: {version}")
    for line in result.stderr.splitlines():      
        if ":" in line:
            _, config_file = line.split(":", 1)
            config_file = config_file.strip()  # Remove leading/trailing whitespace
        print(f"Config File: {config_file}")

    
    if config_file != config_file_linux:
        raise ValueError(f"Invalid config file: {config_file}")
    if not version_pattern.match(version):
        raise ValueError(f"Invalid version: {version}")

    print (" ✅ Verified version and config file succesfully! ")


if __name__ == '__main__':

    host, user, key_filename, machine_name = check_env_vars()
    remote_host = Host(host_ip=host,
                       username=user,
                       key_file_path=key_filename)    
    if "linux" in machine_name.lower():
        run_test_linux(remote_host)

