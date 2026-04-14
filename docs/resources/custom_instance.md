# bluelobster_custom_instance

Creates a custom Blue Lobster instance pinned to a specific host.

## Arguments

- `name` required
- `instance_type` required
- `host` required
- `cores` required
- `memory_size` required
- `disk_size` required
- `gpu_count_input` required
- `gpu_model_input` required
- `username` required
- `template_name` optional
- `iso_url` optional
- `metadata` optional map of strings
- `ssh_public_key_wo` optional write-only credential
- `password_wo` optional write-only credential
- `power_state` optional, `running` or `stopped`
