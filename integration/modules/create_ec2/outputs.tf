output "machine_name"{
  value = var.AWS_MACHINE
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

output "user_name" {
  value =  local.AWS_MACHINE_CONFIGS[var.AWS_MACHINE].default_user
}

output "private_key_path" {
  value = var.PRIVATE_KEY_PATH  
}

output "public_ssh_link" {
  value =  "ssh -i ${var.PRIVATE_KEY_PATH} ${local.AWS_MACHINE_CONFIGS[var.AWS_MACHINE].default_user}@${aws_instance.observe_agent_instance.public_ip}"
  
}