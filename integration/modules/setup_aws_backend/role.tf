# Create an IAM role
# 
# This role will allow:
#   - reading and writing to S3 state bucket, restricted to a specific prefix
#   - reading and writing to a single SecretsManager secret
#
data "aws_iam_policy_document" "assume_role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::384876807807:root"]
    }
  }
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::989541196905:root"]
    }
  }
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::460044344528:root"]
    }
  }
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::767397788203:role/OrganizationAccountAccessRole"]
    }
  }

  #Root Blunderome (460044344528)
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = ["arn:aws:iam::460044344528:oidc-provider/token.actions.githubusercontent.com"]
    }
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values = [
        # lock to repo, but match any branch
        "repo:observeinc/observe-agent:*",
      ]
    }
  }

  #Nikhil-PS Account (767397788203)
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = ["arn:aws:iam::767397788203:oidc-provider/token.actions.githubusercontent.com"]
    }
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values = [
        # lock to repo, but match any branch
        "repo:observeinc/observe-agent:*",
      ]
    }
  }

}

data "aws_iam_policy_document" "s3_access" {
  statement {
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
    ]

    resources = [
      "arn:aws:s3:::observe-agent-terraform-state/*",
    ]
  }
  statement {
    actions = [
      "s3:ListBucket",
    ]
    resources = [
      "arn:aws:s3:::observe-agent-terraform-state",
    ]
  }
}


data "aws_iam_policy_document" "network_sg_ec2" {
  statement {
    actions = [
      "ec2:*",
    ]
    resources = [
      "*",
    ]
    condition {
      test     = "ForAnyValue:StringEquals"
      variable = "ec2:Region"
      values   = ["us-west-1", "us-west-2"]
    }
  }

}


resource "aws_iam_role" "access" {
  name = "gh-observe_agent-repo"
  #path        = var.iam_path
  description         = <<-EOF
    Role for github terraform account access for observe-agent repo
  EOF
  assume_role_policy  = data.aws_iam_policy_document.assume_role.json
  managed_policy_arns = []
  inline_policy {
    name   = "terraform_backend"
    policy = data.aws_iam_policy_document.s3_access.json
  }
  inline_policy {
    name   = "network_sg_ec2"
    policy = data.aws_iam_policy_document.network_sg_ec2.json
  }

}
