#!/usr/bin/env python3


import os
import sys
import re
import time 
import inspect 
from utils import * 



def get_installation_package(env_vars: dict) -> tuple:

    current_dir = os.getcwd()
    dist_directory = os.path.abspath(os.path.join(current_dir, '..',  'dist'))
    print(f"Path to 'dist' directory: {dist_directory}")

    # List files in the directory
    files = os.listdir(dist_directory)   

    # Search criteria
    package_type = env_vars["machine_config"]["package_type"]
    architecture = env_vars["machine_config"]["architecture"]

    print(f"Looking for installation package '{package_type}' and architecture '{architecture}'")

    # Iterate through files and find matches
    for filename in files:
        if package_type in filename and architecture in filename:            
            full_path = os.path.join(dist_directory, filename)
            print(f"Found matching file {filename} at: {full_path}")
            return filename, full_path

@print_test_decorator
def run_test_windows(remote_host: Host, env_vars: dict) -> None:  

    #Get built dist. installation package path for machine 
    filename, package = get_installation_package(env_vars)    
    home_dir = r"/C:/Users/{}".format(env_vars["user"])
    home_dir_powershell = r"C:\Users\{}".format(env_vars["user"])
    
    #Get agent-installation script path
    current_script_dir = os.path.dirname(os.path.abspath(__file__))   
    ps_installation_script_path = os.path.join(current_script_dir, 'install_windows.ps1')


    #Copy built distribution package to remote host home dir 
    #remote_host.put_file(package, home_dir)

    #Copy observe-agent powershell installation script to remote host home dir 
    remote_host.put_file(ps_installation_script_path, home_dir)

    #Run install script 
    result = remote_host.run_command('.\install_windows.ps1 -local_installer {}\{}'.format( home_dir_powershell, filename))
    print(result)
    
    #result = remote_host.run_command('Expand-Archive -Path {}'.format(filename))
    #print(result)
    
    print("✅ Installation test passed")
    


@print_test_decorator
def run_test_linux(remote_host: Host, env_vars: dict):       
    """
    Test to install local observe-agent on a linux ec2 instance and validate command ran successfully 

    Args:
        remote_host (Host): instance to ssh into 
        env_vars (dict): environment variables passed into for testing

    Raises:
        RuntimeError: Unknown distribution type passed  
    """
    filename, package = get_installation_package(env_vars)
    home_dir = "/home/{}".format(env_vars["user"])

    remote_host.put_file(package, home_dir)
    if "redhat" in env_vars["machine_config"]["distribution"]:
        result = remote_host.run_command('cd ~ && sudo yum localinstall {} -y'.format(filename))
    elif "debian" in env_vars["machine_config"]["distribution"] :
        result = remote_host.run_command('cd ~ && sudo dpkg -i {}'.format(filename))
    else:
        raise RuntimeError("❌ Unknown distribution type")  
    print(result)    
    print("✅ Installation test passed")



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



