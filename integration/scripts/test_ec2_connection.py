#!/usr/bin/env python3


import os
import sys
import re
import time 
from utils import *

@print_test_decorator
def run_test_windows(remote_host: Host, env_vars: dict) -> None:  

    """
    This test validates that the UserdataExecution.log finished successfully 
    and ec2 instance is in stable state prior to running other


    Args:
        remote_host (Host): instance to ssh into 
        env_vars (dict): environment variables passed into for testing

    Raises:
        RuntimeError: Failed to verify UserdataExecution.log or agent.logfile
    """
    
  
    tmp_file = "/tmp/UserdataExecution.log"
    cloud_init_file_timeout = 240 # 4 minutes    
    
    if "2022" in env_vars["machine_name"]: #Windows 2022 -  Test windows cloud-init file finished successfully
        print("Windows 2022 detected")
        cloud_init_file = r'/C:/ProgramData/Amazon/EC2Launch/log/agent.log'
      
        for _ in range(cloud_init_file_timeout):        
            remote_host.get_file(cloud_init_file, tmp_file) # This command will automatically test connection 
            with open(tmp_file) as file: #No encoding for windows 2022 needed 
                content = file.read().lower()
                if "script execution finished successfully"  in content:
                    print(" ✅ Verified agent.log had completed successfully!")
                    return 
                else:
                    print(" Looking for the agent.log file to finish completing...")
            time.sleep(1)        
        raise RuntimeError("❌ The agent.log file did not finish successfully in time")  
    else: # Windows 2016/2019 -   Test windows cloud-init file finished successfully
        print("Windows 2016 or 2019 detected")
        cloud_init_file = r'/C:/ProgramData/Amazon/EC2-Windows/Launch/Log/UserdataExecution.log'

        for _ in range(cloud_init_file_timeout):        
            remote_host.get_file(cloud_init_file, tmp_file) # This command will automatically test connection 
            with open(tmp_file, encoding="utf-16") as file:
                content = file.read().lower()
                if "user data script completed"  in content:
                    print(" ✅ Verified UserdataExecution had completed successfully!")
                    return 
                else:
                    print(" Looking for the UserdataExecution.log file to finish completing...")
            time.sleep(1)        
        raise RuntimeError("❌ The UserdataExecution file did not finish successfully in time")  



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
        

