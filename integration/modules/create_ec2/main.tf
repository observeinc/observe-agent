# locals {
#   compute_instances = { for key, value in var.AWS_MACHINE_CONFIGS :
#   key => value if contains(var.AWS_MACHINE_FILTER, key) || length(var.AWS_MACHINE_FILTER) == 0 }
# }



#Reference to which AWS_MACHINE_FILTER will be used for testing 
locals {
  compute_instances = {
    for key, value in var.AWS_MACHINE_CONFIGS :
    key => value
    if(var.AWS_MACHINE_FILTER) == null || key == var.AWS_MACHINE_FILTER
  }
}



# EC2 instance for linux host 
resource "aws_instance" "observe_agent_instance" {
  for_each = local.compute_instances

  ami           = each.value.ami_id
  instance_type = each.value.ami_instance_type

  associate_public_ip_address = true

  subnet_id = data.aws_subnet.subnet_public.id

  vpc_security_group_ids = [data.aws_security_group.ec2_public.id]
  key_name               = data.aws_key_pair.observe_agent_instance.key_name

  user_data         = coalesce(var.USERDATA, file(join("/", ["${path.module}", each.value.user_data])))
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