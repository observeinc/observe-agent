resource "null_resource" "main" {
  provisioner "local-exec" {
    command = <<EOF
            echo "This root module does nothing and is intended for testing purposes for terraform test commands"
            echo "Please call terrafrom test -verbose to run tests from this location" 
            EOF
  }
}
