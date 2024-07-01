//provider "aws" {}


run "setup_ec2" {
  module {
    source = "./modules/create_ec2"
  }
}


run "setup_observe_variables" {
  module {
    source = "./modules/setup_observe_variables"
  }
}



run "test_ec2_connection" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "python3 ./scripts/test_ec2_connection.py"
    env_vars = {
      HOST           = run.setup_ec2.public_ip
      USER           = run.setup_ec2.user_name
      KEY_FILENAME   = run.setup_ec2.private_key_path
      PASSWORD       = run.setup_ec2.password
      MACHINE_NAME   = run.setup_ec2.machine_name
      MACHINE_CONFIG = run.setup_ec2.machine_config
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Error in Check EC2 Connection"
  }
}




run "test_installation" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "python3 ./scripts/test_installation.py"
    env_vars = {
      HOST           = run.setup_ec2.public_ip
      USER           = run.setup_ec2.user_name
      KEY_FILENAME   = run.setup_ec2.private_key_path
      PASSWORD       = run.setup_ec2.password
      MACHINE_NAME   = run.setup_ec2.machine_name
      MACHINE_CONFIG = run.setup_ec2.machine_config
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Error in Agent Installation"
  }
}




run "test_version" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "python3 ./scripts/test_version.py"
    env_vars = {
      HOST           = run.setup_ec2.public_ip
      USER           = run.setup_ec2.user_name
      KEY_FILENAME   = run.setup_ec2.private_key_path
      PASSWORD       = run.setup_ec2.password
      MACHINE_NAME   = run.setup_ec2.machine_name
      MACHINE_CONFIG = run.setup_ec2.machine_config
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Error in Check Version Test"
  }
}




run "test_diagnosis" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "python3 ./scripts/test_diagnosis.py"
    env_vars = {
      OBSERVE_URL    = run.setup_observe_variables.OBSERVE_URL
      OBSERVE_TOKEN  = run.setup_observe_variables.OBSERVE_TOKEN
      HOST           = run.setup_ec2.public_ip
      USER           = run.setup_ec2.user_name
      KEY_FILENAME   = run.setup_ec2.private_key_path
      PASSWORD       = run.setup_ec2.password
      MACHINE_NAME   = run.setup_ec2.machine_name
      MACHINE_CONFIG = run.setup_ec2.machine_config
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Error in Check Diagnosis Test"
  }
}

run "test_status" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "python3 ./scripts/test_status.py"
    env_vars = {
      HOST           = run.setup_ec2.public_ip
      USER           = run.setup_ec2.user_name
      KEY_FILENAME   = run.setup_ec2.private_key_path
      PASSWORD       = run.setup_ec2.password
      MACHINE_NAME   = run.setup_ec2.machine_name
      MACHINE_CONFIG = run.setup_ec2.machine_config
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Error in Check Status Test"
  }
}

