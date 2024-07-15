#!/bin/bash
yum update -y

yum install curl -y

yum install wget -y

yum install ca-certificates -y

sudo yum install -y docker
sudo service docker start
sudo usermod -a -G docker ec2-user
chkconfig docker on