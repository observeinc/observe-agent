## setup_aws_net_sg module

This module sets up the AWS networking and security groups for the integration tests. It's meant to be only setup once for integration tests to use when creating and destroying EC2 instances

The modules uses the IAM role `gh-observe_agent-repo` that is set up from `sertup_aws_backend` module to perform the following actions:

### Networking 
- Create VPC
- Create an Internet Gateway attached to VPC 
- Create route table on our VPC 
- Create a subnet within our VPC
- Create a route table association to the subnet 


### Security Groups
- Create a security group that allows inbound traffic from anywhere on port 22 (SSH) and 3389 (RDP)


The above components will referenced as data sources by the `create_ec2` module for use in terraform tests 

## State 

The state of the this module is stored in the S3 bucket `observe-agent-terraform-state` that was created by the `setup_aws_backend` module. It uses an S3 backend
The role assumption needed for both the S3 backend and AWS provider is the same: `gh-observe_agent-repo` 
