#!/usr/bin/env python3
import os
import sys

def die(message):
    print(message, file=sys.stderr)
    sys.exit(1)

public_ssh_link = "ssh -i ./test_key.pem ec2-user@54.151.114.231"

if not public_ssh_link:
    die("PUBLIC_SSH_LINK not set")

print(f"PUBLIC_SSH_LINK is set to: {public_ssh_link}")
