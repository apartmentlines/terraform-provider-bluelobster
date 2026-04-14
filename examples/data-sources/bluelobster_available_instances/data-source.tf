data "bluelobster_available_instances" "all" {}

output "available_instance_ids" {
  value = [for item in data.bluelobster_available_instances.all.items : item.id]
}

output "available_instance_regions" {
  value = {
    for item in data.bluelobster_available_instances.all.items :
    item.id => [for region in item.regions_with_capacity_available : region.name]
  }
}
