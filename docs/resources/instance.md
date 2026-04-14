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
