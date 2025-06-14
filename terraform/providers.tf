terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }

  cloud {
    organization = "loco-deploy"

    workspaces {
      name = "loco"
    }
  }

}

provider "digitalocean" {
  token = var.do_token
}

