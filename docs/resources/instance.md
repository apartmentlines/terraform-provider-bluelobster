# bluelobster_instance

Creates a standard Blue Lobster instance using `region` and `instance_type`.

## Arguments

- `region` required
- `instance_type` required
- `username` required
- `name` optional
- `template_name` optional
- `iso_url` optional
- `metadata` optional map of strings
- `ssh_public_key_wo` optional write-only credential
- `password_wo` optional write-only credential
- `power_state` optional, `running` or `stopped`

Set at least one of `ssh_public_key_wo` or `password_wo`.

## Attribute Reference

- `id` Blue Lobster instance ID.
- `host_id` Host identifier reported by the API.
- `ip_address` Primary public IP address.
- `internal_ip` Internal IP address.
- `cpu_cores` Number of vCPUs.
- `memory` Memory size in GiB.
- `storage` Storage size in GiB.
- `gpu_count` Number of GPUs.
- `gpu_model` GPU model description.
- `power_status` Observed power status from the API.
- `created_at` Instance creation timestamp.
- `price_cents_per_hour` Hourly price in cents.
- `team_id` Owning team ID.
- `team_name` Owning team name.
- `access_type` Access type reported by the API.
- `template_display_name` Display name of the selected template, when applicable.
- `os_type` OS type reported by the API.
- `vm_username` Effective VM username reported by the API.
