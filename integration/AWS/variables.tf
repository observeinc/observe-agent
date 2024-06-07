# your local key path (assumes it exists) - this will allow you to access ec2 instances
# tflint-ignore: terraform_naming_convention
variable "PUBLIC_KEY_PATH" {
  description = "Public key path"
  nullable    = true
  type        = string
}

# tflint-ignore: terraform_naming_convention
variable "PRIVATE_KEY_PATH" {
  description = "Private key path"
  nullable    = true
  type        = string
}

#where to deploy
#tflint-ignore: terraform_naming_convention
# variable "REGION" {
#   default     = "us-west-2"
#   description = "Where resources will be deployed"
#   type        = string
# }

# appended to resource names so you can find your stuff
variable "name_format" {
  description = "Common prefix for resource names"
  type        = string
}

# tflint-ignore: terraform_naming_convention
variable "BASE_TAGS" {
  description = "base resource tags"
  type        = map(string)
  default = {
    owner        = "Observe"
    createdBy    = "terraform"
    team         = "content"
    purpose      = "test auto configuration script"
    git_repo_url = "https://github.com/observeinc/linux-host-configuration-scripts"
  }
}

# tflint-ignore: terraform_naming_convention
# variable "USE_BRANCH_NAME" {
#   default     = "main"
#   description = "git repository branch to use"
#   type        = string
# }

# tflint-ignore: terraform_naming_convention
variable "CI" {
  type        = bool
  default     = false
  description = "This variable is set to true by github actions to tell us we are running in ci"
}

# tflint-ignore: terraform_naming_convention
variable "PUBLIC_KEY" {
  description = "This value comes from a variable in github actions"
  nullable    = true
  default     = null
  type        = string
}

# tflint-ignore: terraform_naming_convention
variable "AWS_MACHINE_CONFIGS" {
  type        = map(any)
  description = "variables for supported OS"
  default = {

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
      user_data         = "../user_data/yumbased.sh"
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

# tflint-ignore: terraform_naming_convention
variable "AWS_MACHINE_FILTER" {
  description = "This is used as filter agains AWS_MACHINE_CONFIGS in main.tf - if set to true then all values"
  default     = ["UBUNTU_18_04_LTS"]
  type        = any
}

variable "USERDATA" {
  type     = any
  nullable = true
  default  = null
}