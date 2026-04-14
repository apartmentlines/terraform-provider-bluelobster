# Terraform Blue Lobster Complete Test

This directory exercises every resource and data source exposed by the Blue Lobster Terraform provider.

## What it covers

- Always reads:
  - `bluelobster_available_instances`
  - `bluelobster_templates`
  - `bluelobster_instances`
- Optionally creates:
  - `bluelobster_instance`
- When the standard instance is enabled, also manages:
  - `bluelobster_instance_firewall`
  - `bluelobster_backup_schedule`
  - `bluelobster_instance_ip`
  - `bluelobster_instance` data source for the created instance
  - `bluelobster_instance_backups` data source for the created instance

## Usage

1. Ensure `BLUELOBSTER_API_TOKEN` is exported.
2. Ensure the [Blue Lobster Terraform provider](https://github.com/apartmentlines/terraform-provider-bluelobster) is installable by Terraform.
3. Copy `terraform.tfvars.example` to `terraform.tfvars` and adjust values.
4. Run:

```bash
terraform init
terraform plan
terraform apply
```

By default, `create_standard_instance` is `false`, so a plain `plan` only exercises the read-only data sources. Set `create_standard_instance = true` to exercise the resource paths as well.
