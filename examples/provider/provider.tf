terraform {
  required_providers {
    bluelobster = {
      source = "apartmentlines/bluelobster"
    }
  }
}

provider "bluelobster" {
  api_key = var.bluelobster_api_key
}
