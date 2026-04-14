locals {
  standard_enabled = var.create_standard_instance

  standard_ssh_public_key = local.standard_enabled ? trimspace(file(pathexpand(var.standard_ssh_public_key_path))) : null

  support_enabled = var.manage_supporting_resources && local.standard_enabled
}

data "bluelobster_available_instances" "all" {}

data "bluelobster_templates" "all" {}

data "bluelobster_instances" "all" {}

resource "bluelobster_instance" "standard" {
  count = local.standard_enabled ? 1 : 0

  name          = var.standard_instance_name
  region        = var.standard_region
  instance_type = var.standard_instance_type
  username      = var.standard_username
  template_name = var.standard_template_name

  ssh_public_key_wo = local.standard_ssh_public_key
  power_state       = var.standard_power_state

  metadata = {
    managed_by = "terraform"
    scenario   = "provider-test"
    resource   = "standard-instance"
  }
}

resource "bluelobster_instance_firewall" "standard" {
  count = local.support_enabled ? 1 : 0

  instance_id = bluelobster_instance.standard[0].id
  enabled     = true
  policy_in   = "DROP"
  policy_out  = "ACCEPT"

  rules = [
    {
      type    = "in"
      action  = "ACCEPT"
      proto   = "tcp"
      dport   = "22"
      comment = "SSH"
      enabled = true
    },
    {
      type    = "in"
      action  = "ACCEPT"
      proto   = "tcp"
      dport   = "443"
      comment = "HTTPS"
      enabled = true
    }
  ]
}

resource "bluelobster_backup_schedule" "standard" {
  count = local.support_enabled ? 1 : 0

  instance_id  = bluelobster_instance.standard[0].id
  frequency    = var.backup_frequency
  hour_utc     = var.backup_hour_utc
  day_of_week  = var.backup_day_of_week
  day_of_month = var.backup_day_of_month
}

resource "bluelobster_instance_ip" "standard_extra" {
  count = local.support_enabled ? 1 : 0

  instance_id = bluelobster_instance.standard[0].id
}

data "bluelobster_instance" "standard" {
  count = local.standard_enabled ? 1 : 0

  id = bluelobster_instance.standard[0].id
}

data "bluelobster_instance_backups" "standard" {
  count = local.standard_enabled ? 1 : 0

  instance_id = bluelobster_instance.standard[0].id
}
