# bluelobster_instance_firewall

Manages the full ordered firewall policy and rules for an instance.

This resource reconciles the entire rule list. Rule ordering matters and is preserved from the Terraform configuration.

## Argument Reference

- `instance_id` (Required, Forces replacement) Instance ID whose firewall configuration will be managed.
- `enabled` (Required) Whether the firewall is enabled.
- `policy_in` (Required) Default inbound policy. Must be `ACCEPT` or `DROP`.
- `policy_out` (Required) Default outbound policy. Must be `ACCEPT` or `DROP`.
- `rules` (Optional) Ordered firewall rule list to reconcile in full.

## Nested `rules` Blocks

- `type` (Required) Rule direction. Must be `in` or `out`.
- `action` (Required) Rule action. Must be `ACCEPT`, `DROP`, or `REJECT`.
- `source` (Optional) Source CIDR, IP, or other API-supported source expression.
- `dest` (Optional) Destination CIDR, IP, or other API-supported destination expression.
- `proto` (Optional) Protocol such as `tcp`, `udp`, or `icmp`.
- `dport` (Optional) Destination port or port range.
- `sport` (Optional) Source port or port range.
- `comment` (Optional) Free-form rule comment.
- `enabled` (Optional) Whether the rule is enabled. Defaults to enabled when omitted.

## Attribute Reference

- `id` Resource ID. This is set to the same value as `instance_id`.
- `rules[*].pos` Remote rule position assigned by the API.
