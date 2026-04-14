resource "bluelobster_backup_schedule" "daily" {
  instance_id = bluelobster_instance.worker.id
  frequency   = "daily"
  hour_utc    = 3
}
