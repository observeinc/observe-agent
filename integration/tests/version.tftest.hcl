provider "aws"{}


// provider "aws" {
//   region  = "us-west-1" # Specify the AWS region
//   profile = "blunderdome"
//   assume_role {
//     #role_arn = "arn:aws:iam::767397788203:role/OrganizationAccountAccessRole"
//     role_arn = "arn:aws:iam::767397788203:role/gh-observe_agent-repo"
//   }
// }


run "setup_ec2" {
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
      HOST         = run.setup_ec2.public_ip
      USER         = run.setup_ec2.user_name
      KEY_FILENAME = run.setup_ec2.private_key_path
      MACHINE_NAME = run.setup_ec2.machine_name
      MACHINE_CONFIG = run.setup_ec2.machine_config
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Error in Check EC2 State"
  }
}




run "check_installation" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "python3 ./scripts/check_installation.py"
    env_vars = {
      HOST         = run.setup_ec2.public_ip
      USER         = run.setup_ec2.user_name
      KEY_FILENAME = run.setup_ec2.private_key_path
      MACHINE_NAME = run.setup_ec2.machine_name
      MACHINE_CONFIG = run.setup_ec2.machine_config
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Error in Agent Installation"
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
      HOST         = run.setup_ec2.public_ip
      USER         = run.setup_ec2.user_name
      KEY_FILENAME = run.setup_ec2.private_key_path
      MACHINE_NAME = run.setup_ec2.machine_name
      MACHINE_CONFIG = run.setup_ec2.machine_config
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Error in Check Version Test"
  }
}





