
# Random String for naming
resource "random_string" "output" {
  length  = 6
  special = false
}



#Create Key pair for EC2 instance using Public Key Specified in var.PUBLIC_KEY_PATH
resource "aws_key_pair" "ec2key" {
  key_name   = format(var.name_format, "publicKey_${var.AWS_MACHINE}_${random_string.output.id}")
  public_key = file(var.PUBLIC_KEY_PATH)

  tags = merge(
    local.BASE_TAGS,
    {
      Name = format(var.name_format, "publicKey_${var.AWS_MACHINE}_${random_string.output.id}")
    },
  )

}

#Create EC2 instance for Observe Agent Testing
resource "aws_instance" "observe_agent_instance" {

  ami                         = local.AWS_MACHINE_CONFIGS[var.AWS_MACHINE].ami_id
  instance_type               = local.AWS_MACHINE_CONFIGS[var.AWS_MACHINE].ami_instance_type
  associate_public_ip_address = true
  subnet_id                   = data.aws_subnet.subnet_public.id
  vpc_security_group_ids      = [data.aws_security_group.ec2_public.id]
  key_name                    = aws_key_pair.ec2key.key_name

  user_data         = file(join("/", ["${path.module}", local.AWS_MACHINE_CONFIGS[var.AWS_MACHINE].user_data]))
  get_password_data = can(regex("WINDOWS", var.AWS_MACHINE)) ? true : false
  
  root_block_device {
    volume_size = 100
  }
  tags = merge(
    local.BASE_TAGS,
    {
      Name                 = format(var.name_format, "${var.AWS_MACHINE}_${random_string.output.id}")
      OS_KEY               = "${var.AWS_MACHINE}"
    },
  )

  lifecycle {
    precondition {
      condition     = contains(keys(local.AWS_MACHINE_CONFIGS), var.AWS_MACHINE)
      error_message = "The provided AWS_MACHINE value is not valid. It must be one of the keys in AWS_MACHINE_CONFIGS."
    }
  }

}


