## Integration Tests 

The root of this module location is intended to run integration tests using the terraform test framework. The tests are located at `integration/tests`

The tests are run using the `terraform test -verbose` command 


### Variables 

The tests are run using the following variables. These can be set in the `integration/tests.auto.tfvars` file for local testing. 

```
name_format        = "tf-observe-agent-test-%s"
AWS_MACHINE_FILTER = "AMAZON_LINUX_2023"  #Choose the AWS Machine to run the tests on 
PUBLIC_KEY_PATH  = "./test_key.pub" #Path to Public Key for EC2
PRIVATE_KEY_PATH = "./test_key.pem" #Path to Private Key for EC2
aws_profile = "<observe_profile>
aws_region = "us-west-1"
aws_role_arn = "<observe_service_account_role" ## It's recommended to run this as service account where the `modules/setup_aws_net_sg` have been one time setup as the data block will reference them
``` 
