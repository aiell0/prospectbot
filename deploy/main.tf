provider "aws" {
	region = "us-east-1"
}

resource "aws_instance" "example" {
	ami = "ami-9887c6e7"
	instance_type = "t2.micro"
}
