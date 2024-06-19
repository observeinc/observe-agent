locals {
  machine_config_list = formatlist("%s:%s", keys(local.AWS_MACHINE_CONFIGS[var.AWS_MACHINE]), values(local.AWS_MACHINE_CONFIGS[var.AWS_MACHINE]))
}
output "machine_name"{
  value = var.AWS_MACHINE
}

#Outputs a string version of machine config (since tf tests exec module needs a string)
output "machine_config" {
  value =   join(",", local.machine_config_list)
}

output "user_name" {
  value =  local.AWS_MACHINE_CONFIGS[var.AWS_MACHINE].default_user
}

output "arn" {
  value = aws_instance.observe_agent_instance.arn
}

output "instance_id" {
  value = aws_instance.observe_agent_instance.id
  
}
output "instance_state" {
  value = aws_instance.observe_agent_instance.instance_state
}


output "public_ip" {
  value = aws_instance.observe_agent_instance.public_ip
}


output "private_key_path" {
  value = var.PRIVATE_KEY_PATH  
}

output "public_ssh_link" {
  value =  "ssh -i ${var.PRIVATE_KEY_PATH} ${local.AWS_MACHINE_CONFIGS[var.AWS_MACHINE].default_user}@${aws_instance.observe_agent_instance.public_ip}"
  
}