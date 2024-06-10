
provider "aws" {
  region = "us-west-1"  # Specify the AWS region
  profile = "blunderdome"

  assume_role {
    role_arn = "arn:aws:iam::767397788203:role/OrganizationAccountAccessRole"
  }
}

// run "setup_aws" {
//   module {
//     source = "./modules/setup_aws"
//   }
//   variables {
//     PUBLIC_KEY_PATH  = var.PUBLIC_KEY_PATH
//     PRIVATE_KEY_PATH = var.PRIVATE_KEY_PATH
//     AWS_MACHINE_FILTER = var.AWS_MACHINE_FILTER
//     CI                 = var.CI

//   }

// }



run "check_version" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }
 
  variables {
    command = "python3 ./scripts/check_version.py"
    env_vars = {
      HOST = "54.151.114.231"
      USER = "ec2-user"
      KEY_FILENAME = "./test_key.pem"     
      MACHINE_NAME = "AMAZON_LINUX_2023"
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Error in Check Version Test"
  }
}




// run "check_version_python" {
//   module {
//     source  = "./modules/exec_python"
//   }
 
//   variables {
//     command = "./scripts/check_version.py"
//     env_vars = {
//       PUBLIC_SSH_LINK = "ssh -t -i ./test_key.pem ec2-user@54.151.114.231"
//       #PUBLIC_SSH_LINK = run.setup_aws.ec2.public_ssh_link
//     }
//   }

//   assert {
//     condition     = output.error == ""
//     error_message = "Something Failed"
//   }
// }






# run "create_bucket" {
#   module {
#     source  = "observeinc/collection/aws//modules/testing/s3_bucket"
#     version = "2.9.0"
#   }

#   variables {
#     setup = run.setup
#   }
# }

# run "check" {
#   module {
#     source  = "observeinc/collection/aws//modules/testing/exec"
#     version = "2.9.0"
#   }

#   variables {
#     command = "./scripts/check_bucket_not_empty"
#     env_vars = {
#       SOURCE = run.create_bucket.id
#       OPTS   = "--output json"
#     }
#   }

#   assert {
#     condition     = output.error == "bucket is empty"
#     error_message = "Bucket isn't empty"
#   }
# }