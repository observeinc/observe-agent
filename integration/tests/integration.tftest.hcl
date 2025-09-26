//provider "aws" {}

variables {
  old_version = "v2.5.0"
}


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
    error_message = "Error in EC2 Connection Test"
  }
}


run "test_install" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "python3 ./scripts/test_install.py"
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
    error_message = "Error in Installation Test"
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
    error_message = "Error in Version Test"
  }
}

run "test_configure" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "python3 ./scripts/test_configure.py"
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
    error_message = "Error in Configure Test"
  }
}


run "test_start" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "python3 ./scripts/test_start.py"
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
    error_message = "Error in Start Test"
  }
}


run "test_diagnose" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "python3 ./scripts/test_diagnose.py"
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
    error_message = "Error in Diagnose Test"
  }
}


run "test_upgrade" {
  module {
    source  = "observeinc/collection/aws//modules/testing/exec"
    version = "2.9.0"
  }

  variables {
    command = "python3 ./scripts/test_upgrade.py"
    env_vars = {
      HOST           = run.setup_ec2.public_ip
      USER           = run.setup_ec2.user_name
      KEY_FILENAME   = run.setup_ec2.private_key_path
      PASSWORD       = run.setup_ec2.password
      MACHINE_NAME   = run.setup_ec2.machine_name
      MACHINE_CONFIG = run.setup_ec2.machine_config
      OLD_VERSION    = var.old_version
    }
  }

  assert {
    condition     = output.error == ""
    error_message = "Error in Upgrade Test"
  }
}
