resource "bluelobster_instance_ip" "extra" {
  instance_id = bluelobster_instance.worker.id
}
