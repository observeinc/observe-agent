#Create VPC 
resource "aws_vpc" "vpc_public" {
  cidr_block       = "10.0.0.0/16"
  enable_dns_hostnames = "true"
  tags = merge(
    var.BASE_TAGS,
    { Name = format(var.name_format, "vpc") },
  )
}

# Creating Internet Gateway attached to VPC 
resource "aws_internet_gateway" "gateway_public" { 
  vpc_id   = aws_vpc.vpc_public.id            # vpc_id will be generated after we create VPC
  tags = merge(
    var.BASE_TAGS,
    { Name = format(var.name_format, "gateway") }
  )
}

#Create public subnet within our VPC
resource "aws_subnet" "subnet_public" {
  vpc_id     = aws_vpc.vpc_public.id
  cidr_block = "10.0.0.0/24"

  tags = merge(
    var.BASE_TAGS,
    { Name = format(var.name_format, "subnet") }
  )
}

#Create route table on our VPC 
resource "aws_route_table" "rt_public" {
  vpc_id = aws_vpc.vpc_public.id 

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.gateway_public.id
  }


  tags = merge(
    var.BASE_TAGS,
    { Name = format(var.name_format, "route-table") }
  )
}


#Associate Public subnet with abve Route Table 
resource "aws_route_table_association" "rt_public_to_subnet" {
  subnet_id      = aws_subnet.subnet_public.id
  route_table_id = aws_route_table.rt_public.id 
}