data "bluelobster_instances" "all" {}

output "instance_ids" {
  value = [for instance in data.bluelobster_instances.all.instances : instance.id]
}

output "instance_names" {
  value = [for instance in data.bluelobster_instances.all.instances : instance.name]
}
