
#Map of allowed machines 
locals {
  AWS_MACHINE_CONFIGS = {
    # UBUNTU_22_04_LTS = {
    #   # ami used in testing
    #   ami_instance_type = "t3.small"
    #   ami_id            = "<NEED TO REPLACE WITH AMI in THUNDERDOME"
    #   ami_description   = "Canonical, Ubuntu, 22.04 LTS, amd64 jammy image build on 2023-05-16"
    #   default_user      = "ubuntu"
    #   sleep             = 120
    #   user_data         = "user_data/aptbased.sh"
    # }

    UBUNTU_20_04_LTS = {
      # ami used in testing
      ami_instance_type = "t3.small"
      ami_id            = "ami-0892d3c7ee96c0bf7"
      ami_description   = "Canonical, Ubuntu, 20.04 LTS, amd64 focal image build on 2021-11-29"
      default_user      = "ubuntu"
      sleep             = 120
      user_data         = "user_data/aptbased.sh"
    }

    UBUNTU_18_04_LTS = {
      ami_instance_type = "t3.small"
      ami_id            = "ami-0cfa91bdbc3be780c"
      ami_description   = "Canonical, Ubuntu, 18.04 LTS, amd64 bionic image build on 2022-04-11"
      default_user      = "ubuntu"
      sleep             = 120
      user_data         = "user_data/aptbased.sh"
    }

    AMAZON_LINUX_2 = {
      ami_instance_type = "t3.small"
      ami_id            = "ami-02b92c281a4d3dc79"
      ami_description   = "Amazon Linux 2 Kernel 5.10 AMI 2.0.20220419.0 x86_64 HVM gp2"
      default_user      = "ec2-user"
      sleep             = 60
      user_data         = "user_data/yumbased.sh"
    }

    AMAZON_LINUX_2023 = {
      ami_instance_type = "t3.small"
      ami_id            = "ami-0a2781a262879e465"
      ami_description   = "Amazon Linux 2023 AMI 2023.4.20240528.0 x86_64 HVM kernel-6.1"
      default_user      = "ec2-user"
      sleep             = 60
      user_data         = "user_data/yumbased.sh"
    }

    RHEL_8_4_0 = {
      ami_instance_type = "t3.small"
      ami_id            = "ami-0b28dfc7adc325ef4"
      ami_description   = "Red Hat Enterprise Linux 8 (HVM), SSD Volume Type"
      default_user      = "ec2-user"
      sleep             = 120
      user_data         = "user_data/yumbased.sh"
    }

    CENT_OS_7 = {
      # https://wiki.centos.org/Cloud/AWS
      ami_instance_type = "t3.small"
      ami_id            = "ami-0686851c4e7b1a8e1"
      ami_description   = "CentOS 7.9.2009 x86_64 ami-0686851c4e7b1a8e1"
      default_user      = "centos"
      sleep             = 120
      user_data         = "user_data/yumbased.sh"
    }

    WINDOWS_SERVER_2016_BASE = {
      ami_instance_type = "t3.small"
      ami_id            = "ami-0e87182c1094f2344"
      ami_description   = "Microsoft Windows Server 2016 with Desktop Experience Locale English AMI provided by Amazon"
      default_user      = "Administrator"
      sleep             = 120
      user_data         = "user_data/windows.ps"
    }

    WINDOWS_SERVER_2019_BASE = {
      ami_instance_type = "t3.small"
      ami_id            = "ami-01dc5695dfebe46cc"
      ami_description   = "Microsoft Windows Server 2019 with Desktop Experience Locale English AMI provided by Amazon"
      default_user      = "Administrator"
      sleep             = 120
      user_data         = "user_data/windows.ps"
    }

    WINDOWS_SERVER_2022_BASE = {
      ami_instance_type = "t3.small"
      ami_id            = "ami-091f300417a06d788"
      ami_description   = "Microsoft Windows Server 2022 Full Locale English AMI provided by Amazon"
      default_user      = "Administrator"
      sleep             = 120
      user_data         = "user_data/windows.ps"
    }


  }
}


#Map of tags we'll attach
locals {
  BASE_TAGS = {
    owner        = "Observe"
    createdBy    = "terraform"
    team         = "Product Specialists "
    purpose      = "observe-agent integration tests"
    git_repo_url = "https://github.com/observeinc/observe-agent"
  }
}