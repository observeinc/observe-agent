## Integration Tests 

The root of this module location is intended to run integration tests using the terraform test framework. The tests are located at `integration/tests`

The tests are run using the `terraform test -verbose` command from this folder `observe-agent/integration` 

When the above command is run, the tests in the `integration/tests` directory are ran using the variables provided. The tests are ran in the order of the run blocks provided in `<test>.tftest.hcl` 

Generally a test will do the following for any given EC2 Machine:
- Create a machine using the variables provided below in `us-west-1`
- Run a test using `observeinc/collection/aws//modules/testing/exec` module to accept python scripts located at `integration/tests/scripts` 


### Variables 

The tests are run using the following variables. These can be set in the `integration/tests.auto.tfvars` file for local testing. 

```
name_format        = "tf-observe-agent-test-%s"
AWS_MACHINE= "AMAZON_LINUX_2023"  #Choose the AWS Machine to run the tests on 
PUBLIC_KEY_PATH  = "./test_key.pub" #Path to Public Key for EC2
PRIVATE_KEY_PATH = "./test_key.pem" #Path to Private Key for EC2
``` 

Note: You must also set the provider correctly. We use the following settings:
- Region: `us-west-1`
- Profile: `blunderdome`
- IAM Role Assumed: `gh-observe_agent-repo` 
  - The above role has permissions to create and destroy EC2 instances. See `modules/setup_aws_backend/role.tf` for more details.