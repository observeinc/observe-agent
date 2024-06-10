#!/usr/bin/env python3
import os
import sys
import re
from fabric import Connection 

def die(message):
    print(message, file=sys.stderr)
    sys.exit(1)

host = os.environ.get("HOST")
user = os.environ.get("USER")
key_filename = os.environ.get("KEY_FILENAME")
machine_name=os.environ.get("MACHINE_NAME")

if host is None:
    die("Error: HOST environment variable is not set.")

if user is None:
    die("Error: USER environment variable is not set.")

if key_filename is None:
    die("Error: KEY_FILENAME environment variable is not set.")

if machine_name is None:
    die("Error: MACHINE_NAME environment variable is not set.")



conn = Connection(host=host, user=user, connect_kwargs={"key_filename": key_filename})

def run_test_linux():    
    config_file_linux = '/etc/observe-agent/observe-agent.yaml'
    version_pattern = re.compile(r'^\d+\.\d+\.\d+$')


    result = conn.run('observe-agent version', hide=True)
    
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
        raise ValueError(f"Invalid config file: {config_file}")
    if not version_pattern.match(version):
        raise ValueError(f"Invalid version: {version}")



if "linux" in machine_name.lower():
    run_test_linux()


