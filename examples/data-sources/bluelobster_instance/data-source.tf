variable "instance_id" {
  type = string
}

data "bluelobster_instance" "selected" {
  id = var.instance_id
}

output "instance_summary" {
  value = {
    id           = data.bluelobster_instance.selected.id
    name         = data.bluelobster_instance.selected.name
    region       = data.bluelobster_instance.selected.region
    power_status = data.bluelobster_instance.selected.power_status
    ip_address   = data.bluelobster_instance.selected.ip_address
  }
}
