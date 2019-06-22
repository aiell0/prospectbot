provider "aws" {
  region = "us-east-1"
}
terraform {
  backend "s3" {
    bucket         = "prospectbot-terraform-state"
    key            = "global/s3/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "StateLock"
  }
}

resource "aws_s3_bucket" "terraform_state" {
  bucket = "prospectbot-terraform-state"

  versioning {
    enabled = true
  }

  lifecycle {
    prevent_destroy = true
  }
}

