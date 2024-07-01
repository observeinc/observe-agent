#!/usr/bin/env python3


import os
import sys
import re
import time 
from utils import *

@print_test_decorator
def run_test_windows(remote_host: Host, env_vars: dict) -> None:  
    pass   

@print_test_decorator
def run_test_linux(remote_host: Host, env_vars: dict) -> None:    
    """
    This test validates that the cloud-init file finished successfully 
    and ec2 instance is in stable state prior to running other


    Args:
        remote_host (Host): instance to ssh into 
        env_vars (dict): environment variables passed into for testing

    Raises:
        RuntimeError: Failed to verify cloud-init file
    """

    cloud_init_file = "/var/log/cloud-init-output.log"
    tmp_file = "/tmp/cloud-init-output.log"
    cloud_init_file_timeout = 240 # 4 minutes

    #Test cloud-init file finished successfully
    for _ in range(cloud_init_file_timeout):        
        remote_host.get_file(cloud_init_file, tmp_file) # This command will automatically test connection 
        with open(tmp_file, "r") as file:
            content = file.read().lower()
            if "finished at"  in content:
                print(" ✅ Verified cloud-init file had completed successfully!")
                return 
            else:
               print(" Looking for the cloud-init file to finish completing...")
        time.sleep(1)        
    raise RuntimeError("❌ The cloud-init file did not finish successfully in time")  

if __name__ == '__main__':
    
    env_vars = get_env_vars()
    remote_host = Host(host_ip=env_vars["host"],
                       username=env_vars["user"],
                       key_file_path=env_vars["key_filename"],
                       password=env_vars["password"])    
    
    #Test SSH Connection before starting test of interest 
    remote_host.test_conection(int(env_vars["machine_config"]["sleep"]))   
    

    if "redhat" in env_vars["machine_config"]["distribution"] or "debian" in env_vars["machine_config"]["distribution"]:
        run_test_linux(remote_host, env_vars)
    elif "windows" in env_vars["machine_config"]["distribution"]:
        run_test_windows(remote_host, env_vars)
        

