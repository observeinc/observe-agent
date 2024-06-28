

data "aws_security_group" "ec2_public" {
  name = "tf-observe-agent-test-ec2_sg"
}

data "aws_subnet" "subnet_public" {
  filter {
    name   = "tag:Name"
    values = ["tf-observe-agent-test-subnet"]
  }
}

