#!/usr/bin/env python3


import os
import sys
import re
import time 
from utils import * 





@print_test_decorator
def run_test_linux(remote_host, env_vars):       
    pass 
  


if __name__ == '__main__':
    
    env_vars = get_env_vars()
    remote_host = Host(host_ip=env_vars["host"],
                       username=env_vars["user"],
                       key_file_path=env_vars["key_filename"])    
    
    #Test SSH Connection before starting test of interest 
    remote_host.test_conection(int(env_vars["machine_config"]["sleep"]))   

    if "linux" in env_vars["machine_name"].lower() or "rhel" in env_vars["machine_name"].lower():
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