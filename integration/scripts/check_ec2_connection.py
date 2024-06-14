#!/usr/bin/env python3


import os
import sys
import re
import time 
from utils import Host, check_env_vars, die



def run_test_linux(remote_host):    
    cloud_init_file = "/var/log/cloud-init-output.log"
    tmp_file = "/tmp/cloud-init-output.log"
    connection_timeout = 60
    cloud_init_file_timeout = 60

    #Test SSH Connection 
    remote_host.test_conection(connection_timeout)

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
    
    host, user, key_filename, machine_name = check_env_vars()
    remote_host = Host(host_ip=host,
                       username=user,
                       key_file_path=key_filename)       

    if "linux" in machine_name.lower() or "rhel" in machine_name.lower():
        run_test_linux(remote_host)






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