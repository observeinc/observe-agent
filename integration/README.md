# Integration Tests

The root of this module location is intended to run integration tests using the terraform test framework. The tests are located at `integration/tests`

The tests are run using the `terraform test -verbose` command from this folder `observe-agent/integration`

When the above command is run, the tests in the `tests` directory are ran using the variables provided. The tests are ran in the order of the run blocks provided in `<test>.tftest.hcl`

Generally a test will do the following for any given EC2 Machine:

- Create a machine in a private subnet (no public IP) using the variables provided below
- Run a test using `observeinc/collection/aws//modules/testing/exec` module to accept python scripts located at `scripts`. The scripts SSH to the machine through an EC2 Instance Connect Endpoint in its VPC, using `aws ec2-instance-connect open-tunnel` as the SSH proxy command

## Pre-requisites

Ensure you have the following:

- Built version of the agent (in `observe-agent/dist`) using `go-releaser`
- AWS CLI v2 credentials for the integration test account (e.g. `export AWS_PROFILE=<profile>` and `export AWS_REGION=us-west-2`), with permission to launch instances and open Instance Connect tunnels
- Observe Collection URL & Datastream Token to test with
- Generated Private & public key pair (named `test_key.pub` & `test_key.pem`)
- Terraform variables (see below)

The GitHub Actions role and OIDC provider are created once with `modules/setup_aws_backend`.

**Building the agent**:

```sh
goreleaser release --snapshot --clean --verbose
```

**SSH Key Pairs**:

Generate an RSA key in PEM format (Windows instances need RSA to decrypt the Administrator password):

```sh
ssh-keygen -t rsa -m PEM -f test_key.pem && mv test_key.pem.pub test_key.pub
```

## Terraform Variables

The tests are run using the following variables. These can be set in the `integration/tests.auto.tfvars` file for local testing (it is gitignored).

```terraform
name_format       = "tf-observe-agent-test-%s"
AWS_MACHINE       = "AMAZON_LINUX_2023"  # Choose the AWS Machine to run the tests on
PUBLIC_KEY_PATH   = "./test_key.pub"
PRIVATE_KEY_PATH  = "./test_key.pem"
subnet_id         = "<private subnet in a VPC with an EC2 Instance Connect Endpoint>"
security_group_id = "<security group allowing port 22 from the endpoint>"
OBSERVE_URL       = "https://<TENANT_ID>.collect.observe-staging.com"
OBSERVE_TOKEN     = "<DATASTREAM_TOKEN_TO_TEST_OBSERVE_AGENT_WITH>"
```

## Local Testing (without terraform test)

Any of the python scripts in the `scripts` directory can be run directly against an existing machine by setting the outputs of the `create_ec2` and `setup_observe_variables` modules as environment variables:

```sh
HOST="<private ip>"
SSH_PROXY_COMMAND="aws ec2-instance-connect open-tunnel --instance-id <instance id>"
USER="ubuntu"
KEY_FILENAME="./test_key.pem"
MACHINE_NAME="UBUNTU_22_04_LTS"
MACHINE_CONFIG="<machine_config output of create_ec2>"
OBSERVE_URL=""
OBSERVE_TOKEN=""
PASSWORD="None" # Windows Administrator password for Windows machines
```

```sh
cd integration
python3 scripts/test_install.py
```

To SSH to a machine manually, use the `ssh_link` output of `create_ec2`.

Passwords and tokens are masked in script output by default; set `MASK=False` to print them.
