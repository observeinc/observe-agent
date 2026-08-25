

data "aws_security_group" "ec2_public" {
  name = "tf-observe-agent-test-ec2_sg"
}

data "aws_subnet" "subnet_public" {
  filter {
    name   = "tag:Name"
    values = ["tf-observe-agent-test-subnet"]
  }
}

# Windows Server base AMIs are rotated and deregistered by AWS on a frequent
# cadence, so hardcoded IDs go stale and break instance creation with
# "collecting instance settings: empty result". Resolve the latest published
# image at plan time instead of pinning an ID.
data "aws_ami" "windows_server_2016_base" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["Windows_Server-2016-English-Full-Base-*"]
  }

  filter {
    name   = "state"
    values = ["available"]
  }
}

data "aws_ami" "windows_server_2019_base" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["Windows_Server-2019-English-Full-Base-*"]
  }

  filter {
    name   = "state"
    values = ["available"]
  }
}

data "aws_ami" "windows_server_2022_base" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["Windows_Server-2022-English-Full-Base-*"]
  }

  filter {
    name   = "state"
    values = ["available"]
  }
}

