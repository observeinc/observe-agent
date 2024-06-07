locals {
  compute_instances = { for key, value in var.AWS_MACHINE_CONFIGS :
  key => value if contains(var.AWS_MACHINE_FILTER, key) || length(var.AWS_MACHINE_FILTER) == 0 }
}



#Create Key pair for EC2 instance using Public Key Specified in var.PUBLIC_KEY_PATH
resource "aws_key_pair" "ec2key" {
  key_name   = format(var.name_format, "publicKey")
  public_key = file(var.PUBLIC_KEY_PATH)

  tags = merge(
    var.BASE_TAGS,
    {
      Name = format(var.name_format, "_publicKey")
    },
  )

}



# EC2 instance for linux host 
resource "aws_instance" "linux_host_integration" {
  for_each = local.compute_instances

  ami           = each.value.ami_id
  instance_type = each.value.ami_instance_type

  associate_public_ip_address = true

  subnet_id = aws_subnet.subnet_public.id

  vpc_security_group_ids = [aws_security_group.ec2_public.id]
  key_name               = aws_key_pair.ec2key.key_name

  user_data         = coalesce(var.USERDATA, file(each.value.user_data))
  get_password_data = can(regex("WINDOWS", each.key)) ? true : false

  root_block_device {
    volume_size = 100
  }

  tags = merge(
    var.BASE_TAGS,
    {
      Name                 = format(var.name_format, "_${each.key}_${random_string.output[each.key].id}")
      OS_KEY               = each.key
      OBSERVE_TEST_RUN_KEY = local.test_key_value[each.key]
    },
  )
}