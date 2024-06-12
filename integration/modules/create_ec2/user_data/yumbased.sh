#!/bin/bash
yum update -y

yum install curl -y

yum install wget -y

yum install ca-certificates -y



echo '[fury]
name=Gemfury Private Repo
baseurl=https://yum.fury.io/observeinc/
enabled=1
gpgcheck=0' | sudo tee /etc/yum.repos.d/fury.repo

sudo yum install observe-agent -y