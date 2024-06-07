resource "aws_security_group" "ec2_public" {
  name   = format(var.name_format, "ec2_sg")
  vpc_id = aws_vpc.vpc_public.id  
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow incomming SSH connections"
  }

   ingress {
    from_port   = 3389
    to_port     = 3389
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow incoming RDP connections"
  }  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  tags = merge(
    var.BASE_TAGS,
    {
      Name = format(var.name_format, "_ec2")
    },
  )
}