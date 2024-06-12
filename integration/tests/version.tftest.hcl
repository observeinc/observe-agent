
// provider "aws" {
//   #region  = "us-west-1" # Specify the AWS region
//   region = var.region 
//   profile = "blunderdome"

//   assume_role {
//     role_arn = "arn:aws:iam::767397788203:role/gh-observe_agent-repo"
//   }
// }


// variables {
//   name_format        = var.name_format
//   PUBLIC_KEY_PATH    = var.PUBLIC_KEY_PATH
//   PRIVATE_KEY_PATH   = var.PRIVATE_KEY_PATH
//   AWS_MACHINE_FILTER = var.AWS_MACHINE_FILTER #Test a Single Machine locally 
//   CI                 = var.CI
//   region             = var.region
// }


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



