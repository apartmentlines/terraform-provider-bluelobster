output "available_instance_ids" {
  value = [for item in data.bluelobster_available_instances.all.items : item.id]
}

output "template_names" {
  value = [for item in data.bluelobster_templates.all.templates : item.name]
}

output "visible_instance_ids" {
  value = [for item in data.bluelobster_instances.all.instances : item.id]
}

output "standard_instance_summary" {
  value = try({
    id           = bluelobster_instance.standard[0].id
    name         = bluelobster_instance.standard[0].name
    ip_address   = bluelobster_instance.standard[0].ip_address
    power_status = bluelobster_instance.standard[0].power_status
    data_source  = data.bluelobster_instance.standard[0]
    backups      = data.bluelobster_instance_backups.standard[0].backups
    extra_ip     = bluelobster_instance_ip.standard_extra[0].ip_address
    firewall     = bluelobster_instance_firewall.standard[0].rules
    schedule = {
      frequency    = bluelobster_backup_schedule.standard[0].frequency
      hour_utc     = bluelobster_backup_schedule.standard[0].hour_utc
      day_of_week  = bluelobster_backup_schedule.standard[0].day_of_week
      day_of_month = bluelobster_backup_schedule.standard[0].day_of_month
    }
  }, null)
}
