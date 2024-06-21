#!/usr/bin/env python3
import os
import sys
import re
import time 
import pprint
from utils import *
from collections import defaultdict


@print_test_decorator
def run_test_linux(remote_host: Host, env_vars: dict) -> None:    
       
   start_command='sudo systemctl enable --now observe-agent'
   status_command='observe-agent status'
   metrics_dict = defaultdict(list)


   #Start Observe Agent 
   remote_host.run_command(start_command)
   time.sleep(5)

    #Check Status Command
   result = remote_host.run_command(status_command)
   for line in result.stdout.splitlines():      
        if ":" in line:
            metric, value = line.split(":", 1)
            metric = metric.strip()
            value = value.strip()                    
            metrics_dict[metric].append(value)
        print(line)
    

    #Assertions on metrics
   if metrics_dict["Status"][0] != "Running":
        die("❌ Observe Agent is not active")

   
   print("✅ Observe Agent is active and running without errors!")
        

if __name__ == '__main__':

    env_vars = get_env_vars()
    remote_host = Host(host_ip=env_vars["host"],
                       username=env_vars["user"],
                       key_file_path=env_vars["key_filename"])    
    
    #Test SSH Connection before starting test of interest 
    remote_host.test_conection(int(env_vars["machine_config"]["sleep"]))   

    if "redhat" in env_vars["machine_config"]["distribution"] or "debian" in env_vars["machine_config"]["distribution"]:
        run_test_linux(remote_host, env_vars)

    pass 


