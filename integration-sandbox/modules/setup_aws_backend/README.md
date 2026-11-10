## setup_aws_backend module

One-time setup for the AWS account the integration tests run in. Apply it manually with credentials that can create IAM resources; state is kept locally.

It creates:

- The GitHub Actions OIDC provider (`token.actions.githubusercontent.com`). It is account-wide, so if it already exists, import it with `terraform import aws_iam_openid_connect_provider.github <provider arn>` before applying.
- The IAM role `gh-observe_agent-repo`, assumable only through GitHub OIDC by `repo:observeinc/observe-agent:*`, with permissions to:
  - launch instances and import key pairs in the test region, tagging them only at creation
  - terminate instances, delete key pairs and read Windows passwords only for resources tagged `purpose = observe-agent integration tests`
  - open EC2 Instance Connect Endpoint tunnels to port 22

```sh
export AWS_PROFILE=<profile for the test account>
terraform init && terraform apply
```

Store the `role_arn` output in the `SANDBOX_AWS_ROLE_ARN` repository secret.
