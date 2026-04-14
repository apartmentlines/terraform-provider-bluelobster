# Blue Lobster Provider

The Blue Lobster provider manages instances and closely related control-plane objects through the public API.

## Resources

- `bluelobster_instance` ([reference](resources/instance.md), [example](../examples/resources/bluelobster_instance/resource.tf))
- `bluelobster_instance_firewall` ([reference](resources/instance_firewall.md), [example](../examples/resources/bluelobster_instance_firewall/resource.tf))
- `bluelobster_backup_schedule` ([reference](resources/backup_schedule.md), [example](../examples/resources/bluelobster_backup_schedule/resource.tf))
- `bluelobster_instance_ip` ([reference](resources/instance_ip.md), [example](../examples/resources/bluelobster_instance_ip/resource.tf))

## Data Sources

- `bluelobster_available_instances` ([reference](data-sources/available_instances.md), [example](../examples/data-sources/bluelobster_available_instances/data-source.tf))
- `bluelobster_templates` ([reference](data-sources/templates.md), [example](../examples/data-sources/bluelobster_templates/data-source.tf))
- `bluelobster_instances` ([reference](data-sources/instances.md), [example](../examples/data-sources/bluelobster_instances/data-source.tf))
- `bluelobster_instance` ([reference](data-sources/instance.md), [example](../examples/data-sources/bluelobster_instance/data-source.tf))
- `bluelobster_instance_backups` ([reference](data-sources/instance_backups.md), [example](../examples/data-sources/bluelobster_instance_backups/data-source.tf))

## Provider arguments

- `api_key` sensitive API key
- `base_url` optional API base URL, default `https://api.bluelobster.ai/api/v1`
