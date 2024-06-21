#!/usr/bin/env python3
import os
import sys
import re
import time 
from utils import *


@print_test_decorator
def run_test_linux(remote_host: Host, env_vars: dict) -> None:    

    config_file_linux = '/etc/observe-agent/observe-agent.yaml'
    #Can match 0.2.2-SNAPSHOT-b6e1491 or 0.2.2 
    version_pattern = re.compile(r'^\d+\.\d+\.\d+(-[A-Za-z0-9-]+)?$')
  
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
        raise ValueError(f" ❌ Invalid config file: {config_file}")
    if not version_pattern.match(version):
        raise ValueError(f" ❌ Invalid version: {version}")

    print (" ✅ Verified version and config file succesfully! ")


if __name__ == '__main__':

    env_vars = get_env_vars()
    remote_host = Host(host_ip=env_vars["host"],
                       username=env_vars["user"],
                       key_file_path=env_vars["key_filename"])    
    
    #Test SSH Connection before starting test of interest 
    remote_host.test_conection(int(env_vars["machine_config"]["sleep"]))   

    if "redhat" in env_vars["machine_config"]["distribution"] or "debian" in env_vars["machine_config"]["distribution"]:
        run_test_linux(remote_host, env_vars)

