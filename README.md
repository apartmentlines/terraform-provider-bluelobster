# Terraform Provider for Blue Lobster

This repository contains a Terraform provider for managing Blue Lobster infrastructure through the public API at `https://api.bluelobster.ai/api/v1`.

## Supported objects

- Resources: `bluelobster_instance`, `bluelobster_instance_firewall`, `bluelobster_backup_schedule`, `bluelobster_instance_ip`
- Data sources: `bluelobster_available_instances`, `bluelobster_templates`, `bluelobster_instances`, `bluelobster_instance`, `bluelobster_instance_backups`

Main docs: [Provider docs](docs/index.md)

Examples: [examples/](examples/)

## Provider configuration

```hcl
terraform {
  required_providers {
    bluelobster = {
      source = "apartmentlines/bluelobster"
    }
  }
}

provider "bluelobster" {
  api_key = var.bluelobster_api_key
}
```

If `api_key` is omitted, the provider falls back to `BLUELOBSTER_API_KEY` and then `BLUELOBSTER_API_TOKEN`.

Supported provider arguments:

- `api_key` (optional, sensitive): Blue Lobster API key. Can also be supplied with `BLUELOBSTER_API_KEY` or `BLUELOBSTER_API_TOKEN`.
- `base_url` (optional): Blue Lobster API base URL. Defaults to `https://api.bluelobster.ai/api/v1`. Can also be supplied with `BLUELOBSTER_BASE_URL`.

## Quickstart

```hcl
data "bluelobster_available_instances" "all" {}

resource "bluelobster_instance" "worker" {
  region        = "igl"
  instance_type = "v1_gpu_1x_a5000"
  username      = "ubuntu"
  name          = "ml-worker-1"

  template_name     = "UBUNTU-22-04-NV"
  ssh_public_key_wo = file(pathexpand("~/.ssh/id_ed25519.pub"))
  power_state       = "running"
}
```

## Resource model

- `bluelobster_instance` is the standard region + instance type flow.
- Credentials are write-only on create and are not stored in Terraform state.
- `power_state` is declarative and only supports `running` and `stopped`.
- `region`, `instance_type`, image selection, metadata, and credentials are create-time fields and force replacement.
- Set at least one of `ssh_public_key_wo` or `password_wo`, and one of `template_name` or `iso_url`.

## Intentionally skipped

This provider does not currently implement imperative or low-value control-plane features such as console tickets, task polling data sources, instance stats/log data sources, or destructive backup restore flows.

## Build

```bash
make tidy
make test
make install
```
