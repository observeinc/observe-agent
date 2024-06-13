provider "aws" {  #Explicitly set the provider to Variables 
  region = var.aws_region 
  profile = var.aws_profile

  assume_role {
    role_arn = var.aws_role_arn
  }
}


variables {  #Explicitly set variables for this file 
  name_format        = var.name_format
  PUBLIC_KEY_PATH    = var.PUBLIC_KEY_PATH
  PRIVATE_KEY_PATH   = var.PRIVATE_KEY_PATH
  AWS_MACHINE_FILTER = var.AWS_MACHINE_FILTER 
  CI                 = var.CI
}


run "setup_aws" {
  module {
    source = "./modules/create_ec2"
  }
}


run "check_ec2_connection" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "python3 ./scripts/check_ec2_connection.py"
    env_vars = {
      HOST         = run.setup_aws.ec2[var.AWS_MACHINE_FILTER].public_ip
      USER         = run.setup_aws.ec2[var.AWS_MACHINE_FILTER].user_name
      KEY_FILENAME = "${var.PRIVATE_KEY_PATH}"
      MACHINE_NAME = run.setup_aws.ec2[var.AWS_MACHINE_FILTER].machine_name
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Error in Check EC2 State"
  }
}



run "check_version" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "python3 ./scripts/check_version.py"
    env_vars = {
      HOST         = run.setup_aws.ec2[var.AWS_MACHINE_FILTER].public_ip
      USER         = run.setup_aws.ec2[var.AWS_MACHINE_FILTER].user_name
      KEY_FILENAME = "${var.PRIVATE_KEY_PATH}"
      MACHINE_NAME = run.setup_aws.ec2[var.AWS_MACHINE_FILTER].machine_name
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Error in Check Version Test"
  }
}



