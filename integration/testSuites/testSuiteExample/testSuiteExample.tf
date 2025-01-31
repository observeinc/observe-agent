resource "null_resource" "testSuiteExample" {
  provisioner "local-exec" {
    command = <<EOF
            echo "This module does nothing and is intended for testing purposes for terraform test commands"
            echo "Please call terrafrom test -verbose to run tests from this location" 
            EOF
  }
}