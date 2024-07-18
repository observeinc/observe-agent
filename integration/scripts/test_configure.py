#!/usr/bin/env python3
import os
import sys
import re
import time 
import utils as u

@u.print_test_decorator
def run_test_windows(remote_host: u.Host, env_vars: dict) -> None:  

    """
    Test to validate connection of observe-agent to Observe 

    Args:
        remote_host (Host): instance to ssh into 
        env_vars (dict): environment variables passed into for testing

    Raises:
        ValueError: Something failed with initial config or observe-agent -> observe connection 
    """
 
    init_command='Set-Location "C:\Program Files\Observe\observe-agent"; ./observe-agent init-config --token {} --observe_url {}'.format(env_vars["observe_token"], env_vars["observe_url"])
    diagnose_command='Set-Location "C:\Program Files\Observe\observe-agent"; ./observe-agent diagnose'
    
    #Set up correct config with observe url and token 
    result = remote_host.run_command(init_command)

    #Check diagnose command
    result = remote_host.run_command(diagnose_command)
    observe_val = False
    for line in result.stdout.splitlines():      
        if "Request to test URL responded with response code 200" in line:
            print (" ✅ observe-agent -> observe validation passed! ")
            observe_val = True
            break        
    if not observe_val:
        print(result)
        raise ValueError(f"❌ Failed: observe-agent -> observe validation")
    
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
     
    init_command='init-config --token {} --observe_url {}'.format(env_vars["observe_token"], env_vars["observe_url"])
    diagnose_command='diagnose'

    #Set up correct config with observe url and token 
    result = remote_host.run_command(docker_prefix + ' ' + init_command)

    #Check diagnose command
    result = remote_host.run_command(docker_prefix + ' ' + diagnose_command)
    observe_val = False
    for line in result.stdout.splitlines():      
        if "Request to test URL responded with response code 200" in line:
            print (" ✅ observe-agent -> observe validation passed! ")
            observe_val = True
            break        
    if not observe_val:
        print(result)
        raise ValueError(f"❌ Failed: observe-agent -> observe validation")


    pass

@u.print_test_decorator
def run_test_linux(remote_host: u.Host, env_vars: dict) -> None:    

    """
    Test to validate connection of observe-agent to Observe 

    Args:
        remote_host (Host): instance to ssh into 
        env_vars (dict): environment variables passed into for testing

    Raises:
        ValueError: Something failed with initial config or observe-agent -> observe connection 
    """

    init_command='sudo observe-agent init-config --token {} --observe_url {}'.format(env_vars["observe_token"], env_vars["observe_url"])
    diagnose_command='observe-agent diagnose'

    #Set up correct config with observe url and token 
    result = remote_host.run_command(init_command)

    #Check diagnose command
    result = remote_host.run_command(diagnose_command)
    observe_val = False
    for line in result.stdout.splitlines():      
        if "Request to test URL responded with response code 200" in line:
            print (" ✅ observe-agent -> observe validation passed! ")
            observe_val = True
            break        
    if not observe_val:
        print(result)
        raise ValueError(f"❌ Failed: observe-agent -> observe validation")
   

if __name__ == '__main__':

    env_vars = u.get_env_vars(need_observe=True)
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

    pass 


