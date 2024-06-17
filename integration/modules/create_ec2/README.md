## Create EC2 

This module sets up an EC2 Instance and EC2 Key Pair attached to the instance, for agent integration testsing. 

It takes in the following variables to create EC2 Instance + Key Pair:
- name_format
- AWS_MACHINE
- PUBLIC_KEY_PATH
- PRIVATE_KEY_PATH 


It then generates outputs that can be used in other modules (eg: `tests/*` for terraform test)

### Dependencies

It relies on the existence of the following in `us-west-` 
- Security Group Name: `tf-observe-agent-test-ec2_sg` 
- Subnet Name: `tf-observe-agent-test-subnet`

The above are used to attach to the EC2 Instance the module creates. 


