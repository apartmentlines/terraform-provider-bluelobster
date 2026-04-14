variable "instance_id" {
  type = string
}

data "bluelobster_instance_backups" "selected" {
  instance_id = var.instance_id
}

output "backup_storage" {
  value = data.bluelobster_instance_backups.selected.storage
}

output "backup_count" {
  value = data.bluelobster_instance_backups.selected.total
}
