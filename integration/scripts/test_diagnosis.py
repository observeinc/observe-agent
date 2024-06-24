#!/usr/bin/env python3
import os
import sys
import re
import time 
from utils import *



@print_test_decorator
def run_test_linux(remote_host: Host, env_vars: dict) -> None:    

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
    config_file_linux = '/etc/observe-agent/observe-agent.yaml'

    #Set up correct config with observe url and token 
    result = remote_host.run_command(init_command)
    write = False
    use = False
    for line in result.stdout.splitlines():      
        if "Writing configuration values to {}".format(config_file_linux) in line:
            print (" ✅ init-config: wrote configuration succesfully! ")
            write = True
            break        
    for line in result.stderr.splitlines():      
        if "Using config file: ".format(config_file_linux) in line:
            print (" ✅ init-config: using correct config file! ")
            use = True
            break
    if not write or not use:       
        print(result)
        raise ValueError(f"❌ Something went wrong with init-config")
            

    #Check diagnose command
    result = remote_host.run_command(diagnose_command)
    observe_val = False
    for line in result.stdout.splitlines():      
        if "Request to test URL responded with response code 200" in line:
            print (" ✅ observe-agent -> observe valdation passed! ")
            observe_val = True
            break        
    if not observe_val:
        print(result)
        raise ValueError(f"❌ Failed: observe-agent -> observe validation")
   

if __name__ == '__main__':

    env_vars = get_env_vars(need_observe=True)
    remote_host = Host(host_ip=env_vars["host"],
                       username=env_vars["user"],
                       key_file_path=env_vars["key_filename"])    
    
    #Test SSH Connection before starting test of interest 
    remote_host.test_conection(int(env_vars["machine_config"]["sleep"]))   

    if "redhat" in env_vars["machine_config"]["distribution"] or "debian" in env_vars["machine_config"]["distribution"]:
        run_test_linux(remote_host, env_vars)

    pass 


