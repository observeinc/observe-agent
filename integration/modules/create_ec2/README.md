 ## Create EC2 

This module sets up an EC2 Instance and EC2 Key Pair attached to the instance, for agent integration testing. 

It takes in the following variables to create EC2 Instance + Key Pair:
- name_format
- AWS_MACHINE
- PUBLIC_KEY_PATH
- PRIVATE_KEY_PATH
- subnet_id
- security_group_id
- run_id (optional, tags the instance and key pair for cleanup)

It then generates outputs that can be used in other modules (eg: `tests/*` for terraform test)

### Dependencies

The instance gets no public IP. It is reached over SSH through an EC2 Instance Connect Endpoint in its VPC (see the `ssh_proxy_command` output), so:
- `subnet_id` must be in a VPC with an EC2 Instance Connect Endpoint and outbound internet access (package repos, GitHub releases, Observe)
- `security_group_id` must allow port 22 from the endpoint
