data "aws_caller_identity" "current" {}

# The GitHub OIDC provider is account-wide; import it instead if it already exists.
resource "aws_iam_openid_connect_provider" "github" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["6938fd4d98bab03faadb97b34396831e3780aea1", "1c58a3a8518e8759bf075b76b750d4f2df264fcd"]
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github.arn]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = [var.github_subject]
    }
  }
}

# The account is shared, so destructive actions are limited to resources the tests tagged.
data "aws_iam_policy_document" "integration_tests" {
  statement {
    sid = "CreateTestResources"
    actions = [
      "ec2:Describe*",
      "ec2:RunInstances",
      "ec2:ImportKeyPair",
    ]
    resources = ["*"]
    condition {
      test     = "StringEquals"
      variable = "aws:RequestedRegion"
      values   = [var.region]
    }
  }

  statement {
    sid       = "TagOnlyAtCreation"
    actions   = ["ec2:CreateTags"]
    resources = ["*"]
    condition {
      test     = "StringEquals"
      variable = "ec2:CreateAction"
      values   = ["RunInstances", "ImportKeyPair"]
    }
  }

  statement {
    sid       = "ManageTaggedTestResources"
    actions   = ["ec2:TerminateInstances", "ec2:DeleteKeyPair", "ec2:GetPasswordData"]
    resources = ["*"]
    condition {
      test     = "StringEquals"
      variable = "aws:ResourceTag/purpose"
      values   = [var.test_resource_tag]
    }
  }

  statement {
    sid       = "SSHThroughInstanceConnectEndpoint"
    actions   = ["ec2-instance-connect:OpenTunnel"]
    resources = ["arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:instance-connect-endpoint/*"]
    condition {
      test     = "NumericEquals"
      variable = "ec2-instance-connect:remotePort"
      values   = ["22"]
    }
  }
}

resource "aws_iam_role" "access" {
  name               = var.role_name
  description        = "GitHub Actions role for observe-agent integration tests"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

resource "aws_iam_role_policy" "integration_tests" {
  name   = var.policy_name
  role   = aws_iam_role.access.id
  policy = data.aws_iam_policy_document.integration_tests.json
}

output "role_arn" {
  value = aws_iam_role.access.arn
}
