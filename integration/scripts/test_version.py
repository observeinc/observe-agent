#!/usr/bin/env python3
import os
import sys
import re
import time 
import utils as u


@u.print_test_decorator
def run_test_windows(remote_host:u.Host, env_vars: dict) -> None:  

    """
    Test to validate observe-agent version and config file loaded is correct 

    Args:
        remote_host (Host): instance to ssh into 
        env_vars (dict): environment variables passed into for testing

    Raises:
        ValueError: if version or config file is invalid
    """

    config_file_windows = 'C:\\Program Files\\Observe\\observe-agent\\observe-agent.yaml'
    #Can match 0.2.2-SNAPSHOT-b6e1491 or 0.2.2 
    version_pattern = re.compile(r'^\d+\.\d+\.\d+(-[A-Za-z0-9-]+)?$')

    result = remote_host.run_command('Set-Location "${Env:Programfiles}\\Observe\\observe-agent"; ./observe-agent version')    
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

     
    if config_file != config_file_windows:
        raise ValueError(f" ❌ Invalid config file: {config_file}")
    if not version_pattern.match(version):
        raise ValueError(f" ❌ Invalid version: {version}")

    print (" ✅ Verified version and config file succesfully! ")

    pass   


@u.print_test_decorator
def run_test_docker(remote_host: u.Host, env_vars: dict) -> None:  
    docker_prefix='sudo docker run \
        --mount type=bind,source=/proc,target=/hostfs/proc,readonly \
        --mount type=bind,source=/snap,target=/hostfs/snap,readonly \
        --mount type=bind,source=/var/lib,target=/hostfs/var/lib,readonly \
        --mount type=bind,source=/var/log,target=/hostfs/var/log,readonly \
        --mount type=bind,source=/var/lib/docker/containers,target=/var/lib/docker/containers,readonly \
        --mount type=bind,source=$(pwd)/observe-agent.yaml,target=/etc/observe-agent/observe-agent.yaml \
        --pid host \
        $(sudo docker images --format "{{.Repository}}:{{.Tag}}" | grep SNAPSHOT)'
    #docker_prefix='docker run $(docker images --format "{{.Repository}}:{{.Tag}}")'
    config_file_linux = '/etc/observe-agent/observe-agent.yaml'
    version_pattern = re.compile(r'^\d+\.\d+\.\d+(-[A-Za-z0-9-]+)?$')
    home_dir = "/home/{}".format(env_vars["user"])

    #Docker doesn't create a observe-agent.yaml by default so we have to create it, upload to host and let docker
    # mount via $(pwd)/observe-agent.yaml,target=/etc/observe-agent/observe-agent.yaml
    u.create_default_config_file(destination_file_path = "/tmp/observe-agent.yaml")
    remote_host.put_file(local_path="/tmp/observe-agent.yaml", remote_path=home_dir)

    #Run command to get version & config-file info 
    result = remote_host.run_command('{} version'.format(docker_prefix))

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

    env_vars = u.get_env_vars()
    remote_host = u.Host(host_ip=env_vars["host"],
                       username=env_vars["user"],
                       key_file_path=env_vars["key_filename"],
                       password=env_vars["password"])    
    
    #Test SSH Connection before starting test of interest 
    remote_host.test_conection(int(env_vars["machine_config"]["sleep"]))   

    if "redhat" in env_vars["machine_config"]["distribution"] or "debian" in env_vars["machine_config"]["distribution"]:
        run_test_linux(remote_host, env_vars)
    elif "windows" in env_vars["machine_config"]["distribution"]:
        run_test_windows(remote_host, env_vars)
    elif "docker" in env_vars["machine_config"]["distribution"]:
        run_test_docker(remote_host, env_vars)


# docker run --mount type=bind,source=/proc,target=/hostfs/proc,readonly --mount type=bind,source=/snap,target=/hostfs/snap,readonly --mount type=bind,source=/var/lib,target=/hostfs/var/lib,readonly --mount type=bind,source=/var/log,target=/hostfs/var/log,readonly --mount type=bind,source=/var/lib/docker/containers,target=/var/lib/docker/containers,readonly --mount type=bind,source=$(pwd)/observe-agent.yaml,target=/etc/observe-agent/observe-agent.yaml --pid host $(docker images --format "{{.Repository}}:{{.Tag}}")
