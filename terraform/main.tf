terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.16"
    }
  }

  backend "remote" {

    hostname = "app.terraform.io"
    organization = "loco-deploy"
    workspaces {
      name = "loco"
    }
    
  }

  required_version = ">= 1.2.0"
}

provider "aws" {
  region  = "us-east-2"
}
