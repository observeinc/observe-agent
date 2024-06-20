#!/usr/bin/env python3


import os
import sys
import re
import time 
from utils import * 


def get_installation_package(env_vars):

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
def run_test_linux(rremote_host: Host, env_vars: dict):       
    
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
    



if __name__ == '__main__':
    
    env_vars = get_env_vars()
    remote_host = Host(host_ip=env_vars["host"],
                       username=env_vars["user"],
                       key_file_path=env_vars["key_filename"])    
    
    #Test SSH Connection before starting test of interest 
    remote_host.test_conection(int(env_vars["machine_config"]["sleep"]))   

    if "redhat" in env_vars["machine_config"]["distribution"] or "debian" in env_vars["machine_config"]["distribution"]:
        run_test_linux(remote_host, env_vars)







# def die(message):
#     print(message, file=sys.stderr)
#     sys.exit(1)

# host = os.environ.get("HOST")
# user = os.environ.get("USER")
# key_filename = os.environ.get("KEY_FILENAME")
# machine_name=os.environ.get("MACHINE_NAME")

# if host is None:
#     die("Error: HOST environment variable is not set.")

# if user is None:
#     die("Error: USER environment variable is not set.")

# if key_filename is None:
#     die("Error: KEY_FILENAME environment variable is not set.")

# if machine_name is None:
#     die("Error: MACHINE_NAME environment variable is not set.")


# conn = Connection(host=host, user=user, connect_kwargs={"key_filename": key_filename})
# #conn.run()



# #var/log/cloud-init-output.log