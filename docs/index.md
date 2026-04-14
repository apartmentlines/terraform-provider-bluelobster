# Blue Lobster Provider

The Blue Lobster provider manages instances and closely related control-plane objects through the public API.

## Resources

- `bluelobster_instance`
- `bluelobster_custom_instance`
- `bluelobster_instance_firewall`
- `bluelobster_backup_schedule`
- `bluelobster_instance_ip`

## Data Sources

- `bluelobster_available_instances`
- `bluelobster_templates`
- `bluelobster_instances`
- `bluelobster_instance`
- `bluelobster_instance_backups`

## Provider arguments

- `api_key` sensitive API key
- `base_url` optional API base URL, default `https://api.bluelobster.ai/api/v1`
